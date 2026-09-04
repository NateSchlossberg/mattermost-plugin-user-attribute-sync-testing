import {adminStorageStatePath} from './constants';
import {apiDismissOnboarding, apiEnablePlugin, loginAsAdmin} from './utils';

/**
 * Runs once before any test, leaving the server in a known state.
 *
 * Additional setup — extra users, teams, seeded config — belongs here. Prefer the
 * REST API over driving the browser: a setup failure then produces a clear error
 * rather than a screenshot of a half-loaded page.
 */
async function globalSetup() {
    const admin = await loginAsAdmin();

    await admin.storageState({path: adminStorageStatePath});
    await apiEnablePlugin(admin);
    await apiDismissOnboarding(admin);

    await admin.dispose();
}

export default globalSetup;
