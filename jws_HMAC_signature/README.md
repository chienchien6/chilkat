    # JWS HMAC Signature API

    This API provides endpoints to create and validate JWS (JSON Web Signatures) using HMAC SHA-256.

    ## Prerequisites

    - Go installed (with CGO enabled)
    - Chilkat C/C++ library installed and configured (ensure the paths in the CGO directives in `main.go` are correct for your system).
    - MinGW (or a compatible C++ compiler) available in your system's PATH for CGO.

    ## Running the API

    1.  Navigate to the `jws_HMAC_signature` directory:
        ```bash
        cd C:\chilkatPackage\chilkattest\jws_HMAC_signature
        ```
    2.  Run the API server:
        ```bash
        go run .\main.go
        ```
        The server will start on port 8081.

    ## Endpoints

    ### 1. Create JWS (`/create`)

    - **Method:** POST
    - **Request Body (JSON):**
      ```json
      {
        "payload": "Your string payload here",
        "hmacKey": "Your-base64url-encoded-HMAC-key"
      }
      ```
    - **Success Response (JSON):**
      ```json
      {
        "jws": "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.WW91ciBzdHJpbmcgcGF5bG9hZCBoZXJl.signaturePart"
      }
      ```
    - **Error Response (JSON):**
      ```json
      {
        "error": "Error message description"
      }
      ```
    - **Example using curl:**
      ```bash
      curl -X POST -H "Content-Type: application/json" \
      -d '{"payload":"In our village, folks say God crumbles up the old moon into stars.","hmacKey":"AyM1SysPpbyDfgZld3umj1qzKObwVMkoqQ-EstJQLr_T-1qS0gZH75aKtMN3Yj0iPS4hcgUuTwjAzZr1Z9CAow"}' \
      http://localhost:8081/create
      ```

    ### 2. Validate JWS (`/validate`)

    - **Method:** POST
    - **Request Body (JSON):**
      ```json
      {
        "jws": "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.WW91ciBzdHJpbmcgcGF5bG9hZCBoZXJl.signaturePart",
        "hmacKey": "Your-base64url-encoded-HMAC-key"
      }
      ```
    - **Success Response (JSON):**
      ```json
      {
        "isValid": true,
        "payload": "Your string payload here",
        "header": {
          "typ": "JWT",
          "alg": "HS256"
        }
      }
      ```
    - **Invalid Signature Response (JSON):**
      ```json
      {
        "isValid": false,
        "error": "Invalid signature. Key incorrect or JWS modified."
      }
      ```
    - **Error Response (JSON):**
      ```json
      {
        "isValid": false, // May or may not be present depending on error type
        "error": "Error message description"
      }
      ```
    - **Example using curl:**
      ```bash
      curl -X POST -H "Content-Type: application/json" \
      -d '{"jws":"eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.SW4gb3VyIHZpbGxhZ2UsIGZvbGtzIHNheSBHb2QgY3J1bWJsZXMgdXAgdGhlIG9sZCBtb29uIGludG8gc3RhcnMu.bsYsi8HJ0N6OqGI1hKQ9QQRNPxxA5qMpcHLtOvXatk8","hmacKey":"AyM1SysPpbyDfgZld3umj1qzKObwVMkoqQ-EstJQLr_T-1qS0gZH75aKtMN3Yj0iPS4hcgUuTwjAzZr1Z9CAow"}' \
      http://localhost:8081/validate
      ```

    ## Notes
    - The `hmacKey` in the requests must be base64url encoded.
    - The API uses hardcoded algorithm "HS256" and type "JWT" in the header during creation.