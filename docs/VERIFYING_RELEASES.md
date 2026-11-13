# Verifying Tekton CLI Releases

Starting with release v0.x.x, Tekton CLI release artifacts are cryptographically signed using [Sigstore](https://www.sigstore.dev/) cosign for supply chain security.

## Prerequisites

Install cosign to verify the signatures:

```bash
# macOS
brew install cosign

# Linux
wget https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-amd64
chmod +x cosign-linux-amd64
sudo mv cosign-linux-amd64 /usr/local/bin/cosign

# Windows
choco install cosign
```

## Verifying Release Artifacts

Each release includes:
- `checksums.txt` - SHA256 checksums of all release artifacts
- `checksums.txt.sig` - Cosign signature of the checksums file
- `checksums.txt.pem` - Certificate used for signing

### Step 1: Download Release Files

Download the release artifacts from the [GitHub Releases page](https://github.com/tektoncd/cli/releases):

```bash
VERSION=v0.x.x  # Replace with actual version
wget https://github.com/tektoncd/cli/releases/download/${VERSION}/checksums.txt
wget https://github.com/tektoncd/cli/releases/download/${VERSION}/checksums.txt.sig
wget https://github.com/tektoncd/cli/releases/download/${VERSION}/checksums.txt.pem
```

### Step 2: Verify the Signature

Use cosign to verify that the checksums file was signed by the official Tekton CLI release pipeline:

```bash
cosign verify-blob \
  --signature checksums.txt.sig \
  --certificate checksums.txt.pem \
  --certificate-identity-regexp="^https://github.com/tektoncd/cli" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  checksums.txt
```

A successful verification will output:
```
Verified OK
```

### Step 3: Verify Artifact Checksums

After verifying the signature, check that your downloaded artifact matches the checksums:

```bash
# Download the artifact you want
wget https://github.com/tektoncd/cli/releases/download/${VERSION}/tkn_${VERSION}_Linux_x86_64.tar.gz

# Verify checksum (Linux/macOS)
sha256sum --check --ignore-missing checksums.txt

# On macOS, you can also use:
shasum -a 256 -c checksums.txt --ignore-missing
```

## What This Verifies

The signature verification proves:

1. **Authenticity**: The artifacts were produced by the official Tekton CLI release pipeline running in the tektoncd/cli repository
2. **Integrity**: The artifacts have not been tampered with since they were signed
3. **Non-repudiation**: The signature is recorded in the public Sigstore transparency log (Rekor)

## Keyless Signing

Tekton CLI uses **keyless signing** with Sigstore, which means:
- No long-lived signing keys to manage or compromise
- Identity is bound to the GitHub Actions OIDC token from the release workflow
- All signatures are logged in the public Rekor transparency log
- Verification uses the ephemeral certificate included with each release

## Troubleshooting

### Certificate Identity Mismatch

If verification fails with a certificate identity error, ensure you're using the correct identity regexp pattern. The certificate identity will match the GitHub repository URL.

### Network Issues

Cosign requires network access to:
- Rekor transparency log (rekor.sigstore.dev)
- Fulcio certificate authority (fulcio.sigstore.dev)
- TUF root (tuf-repo-cdn.sigstore.dev)

Ensure your network allows outbound HTTPS connections to these services.

### Offline Verification

For offline environments, you can use the `--offline` flag with cosign, but you'll need to fetch the Rekor entry and Fulcio certificate chain beforehand.

## More Information

- [Sigstore Documentation](https://docs.sigstore.dev/)
- [Cosign Documentation](https://docs.sigstore.dev/cosign/overview/)
- [SLSA Framework](https://slsa.dev/)
