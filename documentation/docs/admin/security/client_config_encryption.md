# Encrypt the PMM Client configuration file

The PMM Client configuration file, [`pfw-agent.yaml`](../../use/commands/pmm-agent.md) contains sensitive information like PMM Server credentials and API tokens. By default, this file is stored in plain text, which means that users with read access to the filesystem can see these credentials.

To protect this data, you can encrypt the configuration file so that its contents are unreadable on disk. 

This involves generating an RSA private key and passing it to PMM Client during setup. PGF WatchTower then automatically encrypts the file whenever it saves configuration changes and decrypts it at startup.

Encryption is optional. Without an encryption key, PMM Client continues to read and write the configuration file in plain text.

## Before you start

To encrypt the PMM Client configuration file, you need:

- **PMM Client 3.7.0** or later
- **[OpenSSL](https://docs.openssl.org/master/man1/openssl-genpkey/)** (or any compatible tool) to generate an RSA private key

## How it works

PMM Client uses two layers of encryption to protect the configuration file:

- **AES-256-GCM** encrypts the configuration data and guards against tampering.
- **RSA-OAEP** wraps the AES key so that only your RSA private key can unlock it.

For additional security, you can also protect the RSA private key with a password.

## Set up encryption
To encrypt the PMM Client configuration file, generate an RSA private key and pass it to PMM Client during setup and at startup:
{.power-number}

1. Generate an RSA private key:

    === "Password-protected (recommended)"
        ```bash
        openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 \
          -aes256 -pass env:OPENSSL_PASSWORD \
          -out /etc/pfw-agent-key.pem
        ```

    === "Without password"
        ```bash
        openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 \
          -out /etc/pfw-agent-key.pem
        ```
 
2. Set permissions on the key file:

    ```bash
    chmod 600 /etc/pfw-agent-key.pem
    chown pfw-agent:pfw-agent /etc/pfw-agent-key.pem
    ```

3. Run `pfw-agent setup` with the encryption flags:

    ```bash
    pfw-agent setup \
      --config-file=/opt/postgres1st/watchtower/config/pfw-agent.yaml \
      --server-address=pmm-server.example.com:443 \
      --server-insecure-tls \
      --config-file-key-file=/etc/pfw-agent-key.pem \
      --config-file-key-password="$OPENSSL_PASSWORD" \
      --server-username=admin \
      --server-password=admin
    ```

    If your key is not password-protected, omit `--config-file-key-password`.

4. Start PMM Client with the encryption flags:

    ```bash
    pfw-agent --config-file=/opt/postgres1st/watchtower/config/pfw-agent.yaml \
      --config-file-key-file=/etc/pfw-agent-key.pem \
      --config-file-key-password="$OPENSSL_PASSWORD"
    ```

## Encryption settings

PMM Client accepts encryption settings as either command-line flags or environment variables. Use flags when running `pfw-agent` directly, and environment variables when configuring a service manager like systemd, Docker, or Kubernetes.


| Flag | Environment variable | Description |
|------|---------------------|-------------|
| `--config-file-key-file` | `PFW_AGENT_CONFIG_FILE_KEY_FILE` | Path to the RSA private key file. Required to enable encryption. |
| `--config-file-key-password` | `PFW_AGENT_CONFIG_FILE_KEY_PASSWORD` | Password for the RSA private key. Only needed if the key is password-protected. | 


## Deployment examples

=== "systemd"   

    Create or modify `/etc/systemd/system/pfw-agent.service`:

    ```ini
    [Unit]
    Description=PMM Agent
    After=network.target

    [Service]
    Type=simple
    User=pfw-agent
    Group=pfw-agent

    Environment="PFW_AGENT_CONFIG_FILE_KEY_FILE=/etc/pfw-agent-key.pem"
    # For password-protected keys, use one of the following:
    # Option 1: systemd credentials (systemd 247+)
    # LoadCredential=key_password:/etc/pfw-agent-key-password
    # Option 2: Environment file with restricted permissions
    # EnvironmentFile=-/etc/pfw-agent-encryption.env

    ExecStart=/opt/postgres1st/watchtower/bin/pfw-agent \
    --config-file=/opt/postgres1st/watchtower/config/pfw-agent.yaml

    Restart=on-failure
    RestartSec=10s

    [Install]
    WantedBy=multi-user.target
    ```

=== "Docker/Podman"
    Mount the encryption key as a read-only volume and pass the encryption settings as environment variables:

    ```bash
    docker run -d \
    --name pfw-agent \
    -v /opt/postgres1st/watchtower/config/pfw-agent.yaml:/opt/postgres1st/watchtower/config/pfw-agent.yaml \
    -v /etc/pfw-agent-key.pem:/etc/pfw-agent-key.pem:ro \
    -e PFW_AGENT_CONFIG_FILE_KEY_FILE=/etc/pfw-agent-key.pem \
    -e PFW_AGENT_CONFIG_FILE_KEY_PASSWORD=your-password \
    percona/pmm-client:3 \
    --config-file=/opt/postgres1st/watchtower/config/pfw-agent.yaml
    ```

=== "Kubernetes" 
    Store the encryption key and password in a Kubernetes secret, then reference them in your deployment:

    ```yaml
    apiVersion: v1
    kind: Secret
    metadata:
    name: pfw-agent-encryption-key
    type: Opaque
    data:
    key.pem: <base64-encoded-RSA-key>
    key-password: <base64-encoded-password>
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
    name: pfw-agent
    spec:
    template:
        spec:
        containers:
        - name: pfw-agent
            image: percona/pmm-client:3
            env:
            - name: PFW_AGENT_CONFIG_FILE_KEY_FILE
            value: /etc/encryption/key.pem
            - name: PFW_AGENT_CONFIG_FILE_KEY_PASSWORD
            valueFrom:
                secretKeyRef:
                name: pfw-agent-encryption-key
                key: key-password
            volumeMounts:
            - name: encryption-key
            mountPath: /etc/encryption
            readOnly: true
            - name: config
            mountPath: /opt/postgres1st/watchtower/config/pfw-agent.yaml
            subPath: pfw-agent.yaml
        volumes:
        - name: encryption-key
            secret:
            secretName: pfw-agent-encryption-key
            defaultMode: 0600
        - name: config
            persistentVolumeClaim:
            claimName: pfw-agent-config
    ```

## Migrate from an unencrypted configuration

If PMM Client is already set up, you can enable encryption without re-registering the agent:
{.power-number}

1. Generate an encryption key:

    ```bash
    openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 \
      -aes256 -pass env:OPENSSL_PASSWORD \
      -out /etc/pfw-agent-key.pem
    chmod 600 /etc/pfw-agent-key.pem
    ```

2. Stop PMM Client:

    ```bash
    systemctl stop pfw-agent
    ```

3. Add the encryption environment variables to your [systemd, Docker, or Kubernetes configuration](#deployment-examples).

4. Restart PMM Client to apply the new encryption settings:
    ```bash
    systemctl start pfw-agent
    ```

PMM Client automatically encrypts the configuration file on the next save.

## Disable encryption

To remove encryption and store the configuration file in plain text:
{.power-number}

1. Remove the encryption environment variables (`PFW_AGENT_CONFIG_FILE_KEY_FILE` and `PFW_AGENT_CONFIG_FILE_KEY_PASSWORD`) from your [systemd, Docker, or Kubernetes configuration](#deployment-examples).
2. Restart PMM Client so it can decrypt the file and rewrite it in plain text while the key is still in memory. If you skip this step, the file remains encrypted and PMM Client won't be able to read it on future restarts.

## Verify encryption status

Check whether a configuration file is encrypted by reading it directly:

```bash
# Encrypted: shows binary content, not valid YAML
cat /opt/postgres1st/watchtower/config/pfw-agent.yaml

# You can also confirm with hexdump
head -c 100 /opt/postgres1st/watchtower/config/pfw-agent.yaml | hexdump -C
```

A plain-text file shows readable YAML. An encrypted file shows binary data.

## Key management best practices

- **Back up keys** separately from encrypted configuration files.
- **Use password protection** for RSA private keys.
- **Restrict file permissions** to `0600`, owned by the `pfw-agent` user.
- **Store keys and configuration files in different locations** when possible.
- **Use a secret management system** (HashiCorp Vault, AWS Secrets Manager, etc.) in production environments.
- **Implement key rotation** based on your compliance requirements.
- **Avoid embedding passwords** in scripts or configuration files. Use environment files with restricted permissions or systemd credentials instead.

## Troubleshooting

### "unable to get RSA key from KeyFile"

Check that the key file path is correct and that the file is readable by the `pfw-agent` user. Make sure the file contains a valid RSA private key in PEM format.

### "pkcs8: incorrect password"

Verify that the password is correct and that `PFW_AGENT_CONFIG_FILE_KEY_PASSWORD` (or `--config-file-key-password`) matches the password used to generate the key.

### "unable to RSA-unwrap AES key: crypto/rsa: decryption error"

The configuration file was encrypted with a different key, or the file may be corrupted. Restore from a backup or regenerate the configuration.

### "no valid private key found in a KeyFile"

The key file is not in the correct PEM format or may be corrupted. Regenerate the key file.

## Technical specifications

If you need cryptographic details for security audits or compliance reviews:

- **Encryption**: AES-256-GCM (Galois/Counter Mode), 32-byte key, 12-byte nonce
- **Key wrapping**: RSA-OAEP with SHA-256
- **RSA key size**: 2048 bits minimum, 4096 recommended
- **Key format**: PKCS#8 PEM