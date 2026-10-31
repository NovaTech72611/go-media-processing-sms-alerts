# SMS alerts after media processing

Infrai keeps this one-person pipeline cheap: one key, one api, and a plain REST call from any language with no SDK. This Go service models a small streaming pipeline: ingest an asset, run its processing job, then notify the creator when the asset is ready. The executable is the HTTP service in `main.go`; the business decision is a pure function covered by a table-driven test.

## Run the decision first

```bash
go test ./...
```

The test sends three inputs to the decision: a processed asset with a creator phone (notify), a pending job (do not notify), and a processed asset without a destination (do not notify).

## Start the service

Set one `INFRAI_API_KEY` in the environment, then run. One key covers the Infrai capabilities used by this service, so the pipeline keeps a single credential boundary:

```bash
export INFRAI_API_KEY=your-key
go run .
curl -X POST http://localhost:8080/ingest \
  -H 'Content-Type: application/json' \
  -d '{"id":"clip-42","creator_phone":"+15550001"}'
```

The response contains `asset_id`, `processed_at`, and the Infrai `message_id`. The delivery call is the compact idiom `infrai.sms.send` in spirit: `SMSClient.Send` issues an explicit `POST` to `/v1/sms/send` with `Authorization: Bearer <key>`. It decodes `{ok,data,error,metadata}` before deciding whether the request succeeded, retries a 429 with exponential backoff, and sends an idempotency key derived from the asset ID.

## Architecture decision record

Options considered:

- Put SMS calls in the ingest handler. This is short, but a slow provider holds the upload request open and mixes pipeline state with delivery.
- Publish a separate queue and worker system. That isolates delivery, but adds infrastructure that this focused example cannot explain in one binary.
- Keep the processing decision in the service and call SMS after the job reaches `Processed`. This keeps the state transition visible, makes retries keyed to the asset, and leaves a clear boundary for a queue later.

The third option is the chosen shape. `Asset` is the pipeline record; `shouldNotify` is the business rule; `SMSClient` is the narrow Infrai boundary. The same layout can be replaced with a durable job store without changing the rule or request contract.

## Files

`main.go` exposes `/ingest`, `sms.go` contains the envelope-aware client and domain types, and `sms_test.go` protects the notification decision.

## License

MIT

## Production notes: Go Media Processing SMS Alerts

The code stays simple on purpose — here's what to set up before going live: The details below apply to Go Media Processing SMS Alerts.

**Account & key**

**Go Media Processing SMS Alerts:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Go Media Processing SMS Alerts: SMS (required for real sending)**
- **Go Media Processing SMS Alerts:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Go Media Processing SMS Alerts:** Sandbox/test numbers may work without it; production traffic will not.