# User Attribute Sync Test Tool

A Mattermost plugin that creates plugin-managed user attributes and fills them in from a JSON file. Use it to set up test environments that need attributes the System Console cannot create.

Attributes created by a plugin belong to that plugin. They are marked `protected`, which means only the plugin can change their definition or write their values — admins cannot edit them, through the System Console or the REST API. The `source_only` and `shared_only` access modes are available only on attributes like these.

The plugin creates four attributes in the `access_control` property group, so ABAC policies can reference them as `user.attributes.<name>`. Values come from a JSON file uploaded through the System Console.

This repository is a fork of [mattermost-plugin-user-attribute-sync-starter-template](https://github.com/mattermost/mattermost-plugin-user-attribute-sync-starter-template). If you want a minimal starting point for your own plugin, use that one instead.

## What it creates

| Attribute | Type | Access mode | What it exercises |
|---|---|---|---|
| `job_title` | text | public | Plugin-managed but readable by everyone |
| `programs` | multiselect | `shared_only` | Viewers see only the options they share with the target user |
| `clearance` | rank | `shared_only` | Ordered levels, so policies can say `is at least` |
| `start_date` | date | `source_only` | No one reads it through the API, not even the user themselves |

All four are `protected` and use `visibility: always`. `job_title` is public, so everyone can read it — it is still plugin-managed, and still not editable by an admin.

The attributes appear in **System Console → User Attributes**, on user profiles, and in the ABAC policy editor.

## Requirements

- Mattermost server 12.0.0 or later (the `graph` attribute type requires it)
- An Enterprise Advanced license, for ABAC
- Go 1.26.3 or later, to build the server binaries
- Node v20.11, to build the webapp bundle and run the end-to-end tests

## Install

1. Build and deploy the plugin:

   ```bash
   make deploy
   ```

   Or run `make` and upload `dist/*.tar.gz` through **System Console → Plugin Management**.

2. Go to **System Console → Plugins → User Attribute Sync Test Tool** and set:

   - **Sync Interval (Minutes)** to `1`, so uploads are picked up promptly

   Click **Save**.

3. Edit `data/user_attributes.json` to use the email addresses of users on your server, then upload it with **Choose File** and **Upload** in the same section.

The plugin creates the attributes as soon as it activates. Uploading a file does not trigger a sync, so the values appear on the next sync — within a minute at the interval above.

## The data file

A JSON array of objects, one per user:

```json
[
  {
    "email": "john.doe@example.com",
    "job_title": "Software Engineer",
    "programs": ["Apples", "Oranges"],
    "clearance": "Top Secret",
    "start_date": "2023-01-15"
  }
]
```

`data/user_attributes.json` in this repository is an example of the format. Replace the email addresses with your own test users before uploading it.

- `email` matches the record to a Mattermost user. It is never written as an attribute.
- Every other key is an attribute name from the definitions in `server/sync/field_sync.go`. Keys that do not match a known attribute are skipped with a warning in the logs.
- Text and date values are strings; dates use `YYYY-MM-DD`.
- Multiselect values are an array of option names. Rank and select values are a single option name. Option names are translated to the option IDs Mattermost generated when it created the attribute, so they have to match the definitions exactly.
- Records whose email matches no user are skipped, as are individual values that fail to convert. The rest of the file still syncs.

## Where the values come from

The plugin reads the attributes file from its key-value store, populated by uploading it in the System Console.

### Managing the uploaded file

The settings section shows a panel for the stored file. It reports whether a file is present and when it was uploaded, and lets you replace, download, or delete it. Downloading returns the exact bytes that were uploaded, which is the quickest way to confirm what the plugin is working from. Deleting asks for confirmation.

These buttons act immediately and do not go through the console's **Save** button.

Files are checked before upload: they must be a JSON array of objects and no larger than 10 MB. Individual records are not validated, because one bad record should not stop the rest of the file from syncing.

### Loading data over HTTP

The upload panel is a client for four endpoints, which are useful for seeding a test server from a script. All of them require a system admin.

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/user_attributes` | Store a file. The body is the raw JSON. |
| `GET` | `/user_attributes` | Download the stored file. |
| `GET` | `/user_attributes/status` | `{"exists": bool, "lastUpdated": time\|null}` |
| `DELETE` | `/user_attributes` | Remove the stored file. |

Full paths are prefixed with `/plugins/com.mattermost.user-attribute-sync-test-tool`.

```bash
curl -X POST \
  -H "Authorization: Bearer $MM_ADMIN_TOKEN" \
  --data-binary @data/user_attributes.json \
  http://localhost:8065/plugins/com.mattermost.user-attribute-sync-test-tool/user_attributes
```

## How syncing works

The plugin creates or updates the attribute definitions when it activates. Value sync runs on a background job: immediately on the very first run, then every **Sync Interval (Minutes)** after the previous run finished. The default is 60 minutes and the minimum is 1.

The plugin compares the timestamp written when you uploaded against the timestamp of the last sync; when nothing has changed, the sync does no work.

Uploading a file does not itself trigger a sync. The next scheduled run picks it up.

To sync without waiting, disable and re-enable the plugin. That rebuilds the job, which evaluates the schedule as soon as it starts and runs the sync if the interval has already elapsed since the last run. Note that the last-run time is stored in the plugin's key-value store and survives a restart, so re-enabling part-way through a long interval waits out the remainder — lower the interval first if you are in that position.

Keeping the interval at 1 while you load data avoids the question entirely.

Progress and per-user problems go to the server log:

```bash
make logs-watch
```

## Access modes

The access mode controls who can read an attribute's values through the API and the UI.

**Public** (`job_title`) — everyone can read every value. This is the default when no access mode is set.

**Shared only** (`programs`, `clearance`) — a user sees only the options and values they have in common with the user they are looking at. If Alice is in [Apples, Oranges] and Bob is in [Oranges, Lemons], Alice sees only Oranges on Bob's profile. On a rank attribute, a user sees their own level and lower. Works with select, multiselect, and rank only.

**Source only** (`start_date`) — only this plugin can read the values. Everyone else, including admins, integrations, and the user themselves, sees no value and no options.

`source_only` and `shared_only` require the attribute to be `protected`, which is why they are only available on plugin-managed attributes. These are not LDAP or SAML synced attributes, which are locked to those sync jobs instead.

Access mode is separate from `visibility`, which only controls whether values are shown in the Mattermost UI. Hiding an attribute in the UI does not restrict API access to it.

## Changing which attributes it creates

Attribute definitions live in the `fieldDefinitions` array in `server/sync/field_sync.go`. After editing them, run `make deploy` — the plugin reconciles the definitions on activation.

### Add an attribute

```go
{
    Name:        "department",
    DisplayName: "Department",
    Type:        model.PropertyFieldTypeText,
    AccessMode:  model.PropertyAccessModePublic,
},
```

`Name` is the identifier used as the key in the data file and as `user.attributes.<name>` in ABAC policies, so it must be a valid CEL identifier — no spaces or punctuation. `DisplayName` is the label shown in the UI.

### Change select or multiselect options

```go
{
    Name:        "programs",
    DisplayName: "Programs",
    Type:        model.PropertyFieldTypeMultiselect,
    Options: []model.CustomProfileAttributesSelectOption{
        {Name: "Apples"},
        {Name: "Oranges"},
        {Name: "Lemons"},
        {Name: "Bananas"},
    },
},
```

Mattermost generates an ID for each option and stores values as those IDs. The plugin reads the IDs back and translates names from the data file when it writes values. Existing options are never removed, because users may already hold those values.

### Add a rank attribute

A rank attribute works like a select — a user holds one option — except each option carries an integer that defines the ordering:

```go
{
    Name:        "clearance",
    DisplayName: "Clearance",
    Type:        model.PropertyFieldTypeRank,
    Options: []model.CustomProfileAttributesSelectOption{
        {Name: "CUI", Rank: model.NewPointer(1)},
        {Name: "Confidential", Rank: model.NewPointer(2)},
        {Name: "Secret", Rank: model.NewPointer(3)},
        {Name: "Top Secret", Rank: model.NewPointer(4)},
    },
},
```

The ordering is what lets a policy express a threshold with `is at least` instead of listing every qualifying option, so `user.attributes.clearance >= "Secret"` matches both Secret and Top Secret. Every option on a rank attribute needs a `Rank`, and the plugin refuses to create or update the attribute otherwise.

### Attribute types cannot change

Mattermost does not allow an attribute's type to change after it is created. To change one, delete the attribute — which deletes its values — then update the definition and redeploy.

## Troubleshooting

**Nothing synced.** Check the log with `make logs-watch`. The usual causes are no file uploaded, or emails in the file matching no user on the server. A sync that matches no users logs warnings and still finishes successfully.

**The attributes are read-only in the System Console.** Expected. Every attribute this plugin creates is `protected`, so only the plugin can change its definition or values.

**The plugin fails to activate.** Attribute operations on the `access_control` group need a license. Check that the server has one.

## Limitations

- Attribute definitions are hardcoded in `server/sync/field_sync.go`. The data file supplies values only, so adding or changing an attribute means editing Go and redeploying.
- Uploading a file does not trigger a sync; it happens on the next scheduled run.
- Users are matched by email address only.
- When a file changes, every record in it is synced again. There is no per-record diffing.

## Planned improvements

- Trigger a sync when a file is uploaded, instead of waiting for the next scheduled run.
- Drop the scheduled job entirely, since an upload is the only thing that can change the data.
- Take attribute definitions from the uploaded file, so new attributes can be created without editing Go.

## Development

```text
.
├── server/
│   ├── sync/
│   │   ├── field_sync.go         # Attribute definitions and schema reconciliation
│   │   ├── value_sync.go         # Writing per-user values
│   │   └── kv_store_provider.go  # Reads the uploaded file from the key-value store
│   ├── plugin.go                 # OnActivate / OnDeactivate
│   ├── configuration.go          # Settings
│   ├── http_hooks.go             # The /user_attributes endpoints
│   └── job.go                    # Background sync job
├── webapp/src/
│   ├── index.tsx                 # Registers the custom admin console setting
│   └── components/
│       ├── upload_user_attributes.tsx  # Upload / download / delete panel
│       └── confirm_modal.tsx           # Confirmation dialog for deletion
├── e2e/                          # Playwright tests — see e2e/README.md
└── data/
    └── user_attributes.json      # Example data file
```

```bash
make                  # check-style, test, and build
make test             # Go and webapp unit tests
make check-style      # Linting and type checking
make deploy           # Build and deploy to a running server
make watch            # Rebuild the webapp bundle when its sources change
make deploy-from-watch  # Install the bundle that `make watch` rebuilt
make logs-watch       # Tail the plugin's log on the running server
```

The end-to-end tests need a running server with the plugin deployed, so they are separate from `make test`:

```bash
make test-e2e      # Playwright, against http://localhost:8065 by default
```

See [`e2e/README.md`](e2e/README.md) for prerequisites, how to run a single spec, and the conventions those tests follow.

`make deploy` connects over the server's local mode socket when one is available, which needs local mode enabled in the server's configuration:

```json
{
    "ServiceSettings": {
        "EnableLocalMode": true,
        "LocalModeSocketLocation": "/var/tmp/mattermost_local.socket"
    }
}
```

Without a socket it falls back to the REST API, using `MM_SERVICESETTINGS_SITEURL` with either `MM_ADMIN_TOKEN` or `MM_ADMIN_USERNAME` and `MM_ADMIN_PASSWORD`.

## License

See [LICENSE](LICENSE).
