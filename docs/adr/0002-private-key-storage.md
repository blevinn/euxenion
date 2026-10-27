# ADR 0002: encrypted private-key boundary for offline Xenri

Status: proposed for human security review, scoped to 0.0.5. This ADR describes **offline key storage**, not the v0.1.0 policy/approval boundary.

## Choice and interoperability

Use random 16-byte salt, PBKDF2-HMAC-SHA256 with 600,000 iterations, AES-256-GCM with random 96-bit nonce, and RFC 5208/5958 PKCS#8 DER plaintext inside a versioned JSON envelope. Additional authenticated data binds envelope version, algorithm and deterministic public `kid`. The envelope is a **project-defined serialized format**, not RFC 8018 PBES2 encrypted PKCS#8. Public keys use standard SPKI PEM, independent of private-key storage; external issuers can use their own approved private stores and export PEM.

The PBKDF2 iteration bound on read is 600,000–2,000,000 to avoid both attacker-controlled trivial work factors and denial-of-service work factors. Salt is 16 bytes, nonce 12 bytes; ciphertext and total serialized sizes are bounded before KDF or decoding. `alg` must match the recovered PKCS#8 key and the recomputed `kid`; corrupted or tampered ciphertext fails authentication. Wrong passwords and tampering intentionally have indistinguishable user-facing errors. The CLI never accepts a passphrase in a command-line argument.

## Threat assumptions and limits

At-rest encryption mitigates casual disclosure from lost backups or copied key files, **provided the passphrase has high entropy and is not co-stored**. PBKDF2 is not memory-hard; a short human password is vulnerable to offline brute force if the envelope is stolen. Prefer a randomly generated passphrase from a password manager. Files are owner-only `0600`, written exclusively with no-follow and no overwrite. Reading uses no-follow, verifies that the opened inode matches the checked inode, and checks permissions again after opening. Do not put encrypted or plaintext private keys in git, logs, release assets, trust configs or a server DB. Do not rely on filesystem permissions against privileged adversaries or malware running as the user.

This implementation is currently tested on Linux/macOS-style POSIX filesystems. A `--passphrase-file` path is an explicit automation opt-in, must itself be a non-symlink regular owner-only file and should live outside the checkout. Its bytes are not printed, but may remain accessible to a compromised user account; plaintext passes through process memory during signing. It is not suitable for unattended production secrets without an external secret-manager integration. No implicit reading from environment, arguments or interactive terminal occurs.

## Unsafe operations policy

Unencrypted PKCS#8 imports and exports are **not implemented** in CLI signing paths. There is no plaintext fallback on decryption failure. Unsafe permissions, missing files, unknown envelope fields/duplicate members, malformed KDF settings, symlinks and existing output paths are errors. No automatic overwriting or key rotation occurs; regenerate with a new key identifier and explicitly update the issuer trust store (#11). Private-key revocation must propagate through trusted public-key removal and durable execution authorization later.

## Remaining work

Cross-tool interoperability fixtures and adversarial cryptographic tests belong to #13/#14. Security review should consider migration to a memory-hard KDF (Argon2id) in an explicitly versioned v2 envelope and prompt-based or OS-keychain passphrase handling. Do not claim this guarantees confidentiality against root, malware, or swapping.
