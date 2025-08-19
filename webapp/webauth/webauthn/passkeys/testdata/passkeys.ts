function base64urlToArrayBuffer(base64url: string): ArrayBuffer {
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

function arrayBufferToBase64url(buffer: ArrayBuffer): string {
    const bytes = new Uint8Array(buffer);
    const binary = String.fromCharCode(...bytes);
    const base64 = btoa(binary);

    // Convert standard base64 to base64url
    return base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

type PasskeyCreation = {
    user_handle: string;
    email: string;
    public_key_id?: string;
    error?: string | null;
};



/**
 * Creates a new passkey for a user.
 * @param email - The email address of the user for whom to create the passkey.
 * @param displayName - The username for which to create the passkey.
 */
async function createPasskey(email: string, displayName: string): Promise<PasskeyCreation> {
    console.log("Creating passkey for:", email, displayName);
    try {
        // 1. Fetch registration options from the server
        const response: Response = await fetch('/generate-registration-options', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ "email": email, "display_name": displayName }),
        });

        const options: PublicKeyCredentialCreationOptionsJSON = await response.json();
        console.log("Server responded with options:", options);

        const createOptions = PublicKeyCredential.parseCreationOptionsFromJSON(options)
        console.log("Create Options:", createOptions);
        // 2. Prompt the user to create a new passkey
        const credential = (await navigator.credentials.create(
            {
                publicKey: createOptions
            })) as PublicKeyCredential;

        console.log("Credential created:", credential);
        console.log("Credential id:", credential.id);

        // 3. Send the new credential to the server to be verified and stored
        const attestationResponse = credential.response as AuthenticatorAttestationResponse;

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
                user_handle: options.user.id,
                public_key_id: credential.id,
                email: options.user.name
            };
        } else {
            const error = await verificationResponse.json();
            return {
                user_handle: options.user.id,
                public_key_id: credential.id,
                email: options.user.name,
                error: `failed to register passkey: ${error.message}`
            };
        }
    } catch (err) {
        console.error("Error creating passkey:", err);
        // Need to handle errors correctly:
        // InvalidStateError - programming error
        // NotAllowedError - all other errors.
        return { user_handle: "", email: "", error: `could not create passkey: ${err}` };
    }
}

type PasskeyLogin = {
    success: boolean;
    user_handle: string;
    public_key_id: string;
    error?: string | null;
};

type CredentialRequestResponse = {
    mediation: CredentialMediationRequirement;
    publicKey: PublicKeyCredentialRequestOptionsJSON;
}

/**
 * Authenticates a user with an existing passkey.
 */
async function usePasskey(): Promise<PasskeyLogin> {
    try {
        // 1. Fetch authentication options from the server
        const response = await fetch('/generate-authentication-options');
        const options: CredentialRequestResponse = await response.json();

        console.log("Authentication public key options received:", options.publicKey!);
        console.log("challenge", options.publicKey!.challenge);
        console.log("Mediated authentication public key options received:", options.mediation);

        const publicKey: PublicKeyCredentialRequestOptions = PublicKeyCredential.parseRequestOptionsFromJSON(
            options.publicKey, // JSON-type representation
        );

        // 2. Prompt the user to use their passkey
        const credential = (await navigator.credentials.get({
            mediation: options.mediation,
            publicKey: publicKey
        })) as PublicKeyCredential;
        console.log("Credential received:", credential);

        // 3. Send the assertion to the server for verification
        const assertionResponse = credential.response as AuthenticatorAssertionResponse;
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
            return {
                success: true,
                public_key_id: credential.id,
                user_handle: verificationData.response.userHandle ?? "",
            };
        } else {
            const error = await verificationResponse.json();
            return {
                success: false,
                public_key_id: "",
                user_handle: "",
                error: `Authentication failed: ${error.message}`,
            };
        }
    } catch (err) {
        return {
            success: false,
            user_handle: "",
            public_key_id: "",
            error: `Could not authenticate with passkey: exception: ${err}`,
        };
    }
}