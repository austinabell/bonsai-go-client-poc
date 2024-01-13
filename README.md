# bonsai-sdk-go

NOTE: This is intended as a proof of concept and should _not_ be used as is. Use with great caution.

## Usage

1. Upload current guest to bonsai

```console
cd examples/factors
BONSAI_API_KEY=<api-key> BONSAI_API_URL=https://api.bonsai.xyz cargo run
```

And then pull the image ID from the output to use in the go script (if the program changes).

This only needs to be done once per program when it changes.

2. Generate proofs with custom input

```console
BONSAI_API_KEY=<api-key> go run ./cmd/main.go
```

This will save a receipt to `./receipt.bin`. This can be verified with Rust code or otherwise.

### Generating openapi template code

```console
oapi-codegen -package=bonsai -generate=types,client -o=client.go bonsai-oapi.yml
```

## TODO (all optional)

- [ ] Stark to snark API calls
- [ ] Create stable wrapper API around the oapi codegen (stability and usability)
- [ ] Could verify receipts through FFI to do sanity checks before validating on-chain 
  - currently requires passing receipt to Rust
- [ ] Use and scope out which serialization protocols would be consistent to serialize in Go and deserialize in the guest
  - This currently just manually serializes values based on risc0 serialization for the factors example
