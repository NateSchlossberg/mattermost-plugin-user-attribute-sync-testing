import {act, render, screen} from '@testing-library/react';
import React from 'react';

import UploadUserAttributes from './upload_user_attributes';

let fetchMock: jest.Mock;

beforeEach(() => {
    // The panel asks the server what is stored as soon as it mounts, and jsdom provides no fetch.
    // Answering "no file" keeps that out of the way of these tests.
    fetchMock = jest.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({exists: false, lastUpdated: null}),
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;
});

// Renders the panel and lets its mount-time status request settle, so the state update it triggers
// happens inside act() rather than after the test has finished.
const renderPanel = async (props: Partial<React.ComponentProps<typeof UploadUserAttributes>> = {}) => {
    const result = render(
        <UploadUserAttributes
            id='Attributes'
            {...props}
        />,
    );

    await act(async () => {
        await Promise.resolve();
    });

    return result;
};

test('renders the upload controls', async () => {
    await renderPanel();

    expect(screen.getByRole('button', {name: 'Choose File'})).toBeInTheDocument();
});

test('disables the upload controls when the plugin itself is disabled', async () => {
    await renderPanel({disabled: true});

    expect(screen.getByRole('button', {name: 'Choose File'})).toBeDisabled();
});
