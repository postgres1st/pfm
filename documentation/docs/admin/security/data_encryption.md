# Postgres1st WatchTower data encryption

Postgres1st (Postgres1st WatchTower) implements robust encryption for sensitive data stored in its internal database's `agent` table. This includes access credentials and configuration details.

## Default encryption

Postgres1st WatchTower automatically manages encryption using a key file located at `/srv/pfw-encryption.key`. Postgres1st WatchTower generates this file upon the initial launch of Postgres1st WatchTower 3 or when upgrading from the latest version of Postgres1st WatchTower 2.

## Custom encryption key configuration

For enhanced security control, Postgres1st WatchTower supports custom encryption keys.

**Key format requirements:**

- The key must be a 32-byte (256-bit) random value, suitable for AES-256-GCM encryption.
- The file must contain exactly 32 raw bytes (not a hex-encoded or base64-encoded string).


Postgres1st WatchTower uses this key with the TINK `AES256GCMKeyTemplate` output prefix type.

To set up a custom key, configure the `PMM_ENCRYPTION_KEY_PATH` environment variable to point to your custom key file.

!!! hint alert alert-success "Important"
    Configure this **before** any data encryption occurs: either before upgrading to Postgres1st WatchTower 3 or before initially starting a new Postgres1st WatchTower 3.x instance.

### Key management requirements

Once configured, Postgres1st WatchTower will use the custom key to encrypt and decrypt all sensitive data stored within the system.

If the custom key is unavailable or misplaced, Postgres1st WatchTower will be unable to access and decrypt the stored data, which will prevent it from running correctly.

Make sure to store and manage the custom encryption key securely to avoid potential loss of data access.

## Rotating the encryption key

You may want to generate a new encryption key or rotate it when the original key is compromised or as part of routine security maintenance. For this, you can use the **Postgres1st WatchTower Encryption Rotation Tool**.

This tool re-encrypts all existing sensitive data with a newly generated encryption key, ensuring continuous security with minimal disruption.

To rotate the encryption key:
{.power-number}

1. Log in to the container that runs PMM Server.

2. Run the Encryption Rotation Tool using the following command:

    ```bash
     pfw-encryption-rotation
    ```

    - Ensure `PMM_ENCRYPTION_KEY_PATH` is set to the current custom key if using one, so the tool can decrypt data before re-encryption.
    - If using custom credentials/SSL for the Postgres1st WatchTower internal database, provide them with the appropriate flags.

3. Verify Postgres1st WatchTower functionality all components are functioning properly to ensure that the encryption key rotation was successful.

Once the rotation tool has completed, a new encryption key will be generated and saved either in the default location (`/srv/pfw-encryption.key`) or in the path specified by `PMM_ENCRYPTION_KEY_PATH`. The tool will automatically re-encrypt all sensitive data with the new key.

## Best practices for custom key management

- Always keep a secure backup of your encryption key, especially when using `PMM_ENCRYPTION_KEY_PATH`, as it is critical to Postgres1st WatchTower’s data decryption process.
- In containerized environments, ensure `PMM_ENCRYPTION_KEY_PATH` is persistently set in the container configuration to avoid issues during restarts.
- Test the encryption key rotation process in a staging environment before applying it in production to minimize potential downtime or configuration issues.

## See also

[Encrypt the PMM Client configuration file](client_config_encryption.md)
