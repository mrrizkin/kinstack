type WebAuthnOptions = {
  publicKey: Record<string, unknown>;
};

function decode(value: string): ArrayBuffer {
  const normalized = value
    .replace(/-/g, "+")
    .replace(/_/g, "/")
    .padEnd(Math.ceil(value.length / 4) * 4, "=");
  const bytes = Uint8Array.from(atob(normalized), (character) => character.charCodeAt(0));
  return bytes.buffer;
}

function encode(value: ArrayBuffer): string {
  return btoa(String.fromCharCode(...new Uint8Array(value)))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/g, "");
}

function prepareOptions(
  payload: WebAuthnOptions,
  registration: boolean,
): PublicKeyCredentialCreationOptions | PublicKeyCredentialRequestOptions {
  const publicKey = { ...payload.publicKey } as Record<string, unknown>;
  if (typeof publicKey.challenge === "string") {
    publicKey.challenge = decode(publicKey.challenge);
  }
  if (registration) {
    const user = { ...(publicKey.user as Record<string, unknown>) };
    if (typeof user.id === "string") user.id = decode(user.id);
    publicKey.user = user;
    const excludeCredentials = publicKey.excludeCredentials as Array<Record<string, unknown>> | undefined;
    if (excludeCredentials) {
      publicKey.excludeCredentials = excludeCredentials.map((item) => ({ ...item, id: decode(String(item.id)) }));
    }
  } else {
    const allowCredentials = publicKey.allowCredentials as Array<Record<string, unknown>> | undefined;
    if (allowCredentials) {
      publicKey.allowCredentials = allowCredentials.map((item) => ({ ...item, id: decode(String(item.id)) }));
    }
  }
  return publicKey as unknown as PublicKeyCredentialCreationOptions | PublicKeyCredentialRequestOptions;
}

function serializeCredential(credential: PublicKeyCredential): Record<string, unknown> {
  const response = credential.response;
  const serialized: Record<string, unknown> = {
    id: credential.id,
    rawId: encode(credential.rawId),
    type: credential.type,
    response: {
      clientDataJSON: encode(response.clientDataJSON),
    },
  };
  if (response instanceof AuthenticatorAttestationResponse) {
    serialized.response = {
      ...(serialized.response as Record<string, unknown>),
      attestationObject: encode(response.attestationObject),
      transports: response.getTransports?.() ?? [],
    };
  } else if (response instanceof AuthenticatorAssertionResponse) {
    serialized.response = {
      ...(serialized.response as Record<string, unknown>),
      authenticatorData: encode(response.authenticatorData),
      signature: encode(response.signature),
      userHandle: response.userHandle ? encode(response.userHandle) : null,
    };
  }
  return serialized;
}

export function supportsWebAuthn(): boolean {
  return typeof window !== "undefined" && typeof PublicKeyCredential !== "undefined" && !!navigator.credentials;
}

export async function createPasskey(payload: WebAuthnOptions): Promise<Record<string, unknown>> {
  if (!supportsWebAuthn()) throw new Error("This browser does not support WebAuthn.");
  const credential = await navigator.credentials.create({
    publicKey: prepareOptions(payload, true) as PublicKeyCredentialCreationOptions,
  });
  if (!(credential instanceof PublicKeyCredential)) throw new Error("The authenticator did not return a passkey.");
  return serializeCredential(credential);
}

export async function getPasskey(payload: WebAuthnOptions): Promise<Record<string, unknown>> {
  if (!supportsWebAuthn()) throw new Error("This browser does not support WebAuthn.");
  const credential = await navigator.credentials.get({
    publicKey: prepareOptions(payload, false) as PublicKeyCredentialRequestOptions,
  });
  if (!(credential instanceof PublicKeyCredential)) throw new Error("The authenticator did not return a passkey.");
  return serializeCredential(credential);
}
