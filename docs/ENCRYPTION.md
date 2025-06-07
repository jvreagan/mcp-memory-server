# Encryption at Rest

The MCP Memory Server supports AES-256-GCM encryption for memory files stored on disk, providing an additional layer of security for sensitive data.

## Configuration

Encryption can be enabled via environment variables:

```bash
# Enable encryption (disabled by default)
export MCP_ENABLE_ENCRYPTION=true

# Path to encryption key file (default: ~/.mcp-memory/encryption.key)
export MCP_ENCRYPTION_KEY_PATH=/path/to/encryption.key
```

## How It Works

1. **Key Generation**: When encryption is enabled and no key file exists, a new 256-bit encryption key is automatically generated and saved to the specified path.

2. **Key Storage**: The encryption key is stored with restricted permissions (0600) to ensure only the owner can read it.

3. **Encryption Process**: 
   - Memory data is first serialized to JSON
   - If compression is enabled, the JSON is compressed using gzip
   - The data is then encrypted using AES-256-GCM
   - Each encrypted file includes a unique 12-byte nonce for security

4. **Decryption Process**:
   - The encrypted data is read from disk
   - Data is decrypted using the stored key
   - If compressed, the data is decompressed
   - JSON is deserialized back into memory objects

## Security Considerations

- **Key Protection**: The encryption key file should be protected with appropriate file system permissions
- **Key Backup**: Make sure to backup the encryption key file - without it, encrypted memories cannot be recovered
- **Key Rotation**: Currently, key rotation requires decrypting all memories with the old key and re-encrypting with a new key

## Compatibility

- The reporting server can decrypt memories if configured with the same encryption key path
- Encrypted and unencrypted memories cannot coexist in the same data directory
- Once encryption is enabled, it should remain enabled to access existing memories

## Example Usage

```bash
# Start the server with encryption
export MCP_ENABLE_ENCRYPTION=true
export MCP_ENCRYPTION_KEY_PATH=/secure/location/mcp.key
./mcp-memory-server

# Start the reporting server with the same key
export MCP_ENABLE_ENCRYPTION=true
export MCP_ENCRYPTION_KEY_PATH=/secure/location/mcp.key
./mcp-reporting-server
```

## Performance Impact

Encryption adds minimal overhead:
- Small increase in CPU usage for encryption/decryption operations
- No significant impact on memory usage
- File sizes may be slightly larger due to encryption metadata (nonce + authentication tag)