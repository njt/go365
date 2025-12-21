# Examples

This directory contains example programs demonstrating how to use go365 and the libgo365 library.

## whoami

A simple example that shows how to:
- Load configuration
- Authenticate using stored credentials
- Use the Microsoft Graph API client
- Retrieve current user information

### Building and Running

```bash
# Build the example
cd examples/whoami
go build -o go365-whoami .

# Move to PATH (optional)
sudo mv go365-whoami /usr/local/bin/

# Run as a plugin (if in PATH)
go365 whoami

# Or run directly
./go365-whoami
```

### Prerequisites

Before running this example, you need to:

1. Configure your Azure AD application:
   ```bash
   go365 config set --tenant-id YOUR_TENANT_ID --client-id YOUR_CLIENT_ID
   ```

2. Login to Microsoft 365:
   ```bash
   go365 login
   ```

3. Verify authentication:
   ```bash
   go365 status
   ```

## Creating Your Own Plugins

To create a custom go365 plugin:

1. Create a new Go program that uses the `libgo365` library
2. Build it with a name starting with `go365-` (e.g., `go365-myplugin`)
3. Place the executable in your PATH
4. Run it with `go365 myplugin`

Example structure:

```go
package main

import (
    "context"
    "fmt"
    "github.com/njt/go365/libgo365"
)

func main() {
    // Your plugin logic here
    // Use libgo365 for authentication and API access
}
```

Build and install:

```bash
go build -o go365-myplugin .
sudo mv go365-myplugin /usr/local/bin/
go365 myplugin
```
