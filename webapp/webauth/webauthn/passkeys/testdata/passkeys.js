function base64urlToArrayBuffer(base64url) {
    // Convert base64url to standard base64
    const base64 = base64url
        .replace(/-/g, '+')
        .replace(/_/g, '/')
        .padEnd(Math.ceil(base64url.length / 4) * 4, '=');
    // Decode base64 to binary string
    const binary = atob(base64);
    // Convert binary string to Uint8Array
    const bytes = Uint8Array.from(binary, char => char.charCodeAt(0));
    // Return the underlying ArrayBuffer
    return bytes.buffer;
}
function arrayBufferToBase64url(buffer) {
    const bytes = new Uint8Array(buffer);
    const binary = String.fromCharCode(...bytes);
    const base64 = btoa(binary);
    // Convert standard base64 to base64url
    return base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}
/**
 * Creates a new passkey for a user.
 * @param email - The email address of the user for whom to create the passkey.
 * @param displayName - The username for which to create the passkey.
 */
async function createPasskey(email, displayName) {
    console.log("Creating passkey for:", email, displayName);
    try {
        // 1. Fetch registration options from the server
        const response = await fetch('/generate-registration-options', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ "email": email, "display_name": displayName }),
        });
        const options = await response.json();
        console.log("User:", options.user);
        console.log("Relying Party:", options.rp);
        // Convert challenge/user ID to ArrayBuffers
        const publicKey = {
            pubKeyCredParams: options.pubKeyCredParams,
            rp: options.rp,
            challenge: base64urlToArrayBuffer(options.challenge),
            user: Object.assign(Object.assign({}, options.user), { id: base64urlToArrayBuffer(options.user.id) // as unknown as string)
             })
        };
        // 2. Prompt the user to create a new passkey
        const credential = (await navigator.credentials.create({
            publicKey
        }));
        console.log("Credential:", credential);
        // 3. Send the new credential to the server to be verified and stored
        const attestationResponse = credential.response;
        const verificationData = {
            id: credential.id,
            rawId: arrayBufferToBase64url(credential.rawId),
            type: credential.type,
            response: {
                clientDataJSON: arrayBufferToBase64url(attestationResponse.clientDataJSON),
                attestationObject: arrayBufferToBase64url(attestationResponse.attestationObject),
            },
        };
        console.log("Verification Data:", verificationData);
        const verificationResponse = await fetch('/verify-registration', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(verificationData),
        });
        if (verificationResponse.ok) {
            return {
                user_handle: credential.id,
                public_key_id: options.user.id,
                email: options.user.name
            };
        }
        else {
            const error = await verificationResponse.json();
            return {
                user_handle: credential.id,
                email: options.user.name,
                error: `failed to register passkey: ${error.message}`, exception: null
            };
        }
    }
    catch (err) {
        // Need to handle errors correctly:
        // InvalidStateError - programming error
        // NotAllowedError - all other errors.
        return { user_handle: "", email: "", exception: `could not create passkey: ${err}` };
    }
}
/**
 * Authenticates a user with an existing passkey.
 */
async function usePasskey() {
    try {
        // 1. Fetch authentication options from the server
        const response = await fetch('/generate-authentication-options');
        const options = await response.json();
        // Convert challenge and any credential IDs from Base64URL to ArrayBuffer
        options.challenge = base64urlToArrayBuffer(options.challenge);
        if (options.allowCredentials) {
            for (const cred of options.allowCredentials) {
                cred.id = base64urlToArrayBuffer(cred.id);
            }
        }
        console.log("Authentication options received:", options);
        // 2. Prompt the user to use their passkey
        const credential = (await navigator.credentials.get({
            publicKey: options,
        }));
        console.log("Credential received:", credential);
        // 3. Send the assertion to the server for verification
        const assertionResponse = credential.response;
        console.log("Assertion response:", assertionResponse);
        console.log("Assertion response userHandle:", arrayBufferToBase64url(assertionResponse.userHandle ? assertionResponse.userHandle : new ArrayBuffer(0)));
        const verificationData = {
            id: credential.id,
            rawId: arrayBufferToBase64url(credential.rawId),
            type: credential.type,
            response: {
                clientDataJSON: arrayBufferToBase64url(assertionResponse.clientDataJSON),
                authenticatorData: arrayBufferToBase64url(assertionResponse.authenticatorData),
                signature: arrayBufferToBase64url(assertionResponse.signature),
                userHandle: assertionResponse.userHandle ? arrayBufferToBase64url(assertionResponse.userHandle) : null,
            },
        };
        console.log("verification user handle:", verificationData.response.userHandle);
        const verificationResponse = await fetch('/verify-authentication', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(verificationData),
        });
        if (verificationResponse.ok) {
            // Redirect or update UI for signed-in state
            return "";
        }
        else {
            const error = await verificationResponse.json();
            return `❌ Authentication failed: ${error.message}`;
        }
    }
    catch (err) {
        return `Could not authenticate with passkey: ${err}`;
    }
}
