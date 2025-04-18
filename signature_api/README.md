# Signature API

This is a simple Go API that generates an XML digital signature using Chilkat library.

## Prerequisites

- Go installed (with CGO enabled)
- Chilkat C/C++ library installed and configured (ensure the paths in the CGO directives in `main.go` are correct for your system).
- MinGW (or a compatible C++ compiler) available in your system's PATH for CGO.

## Running the API

1.  Navigate to the `signature_api` directory in your terminal:
    ```bash
    cd signature_api
    ```
2.  (Optional) Place your ECDSA private key in a file named `private-key.zip` in the `signature_api` directory. The zip file should contain a single `.pem` file with the private key.
    - If `private-key.zip` is not found, the API will attempt to download a sample key from `https://www.chilkatsoft.com/exampleData/secp256r1-key.zip`.
3.  Build and run the API server:
    ```bash
    go run .
    ```
    The server will start on port 8080 by default.

## Using the API

Send a GET request to the `/sign` endpoint:

```
http://localhost:8080/sign
```

**Example using curl:**

```bash
curl http://localhost:8080/sign
```

**Successful Response (Status 200 OK):**

The response will be a JSON object containing the generated XML signature:

```json
{
  "signature": "<Signature xmlns=\"http://www.w3.org/2000/09/xmldsig#\">... signature XML content ...</Signature>"
}
```

**Error Response (Status 500 Internal Server Error):**

If an error occurs during the process (e.g., failed to load key, failed to generate signature), the response will be a JSON object with an error message:

```json
{
  "error": "Specific error message here..."
}
```
Error details will also be logged to the server console.
