# Compliance

An app whose `app.yaml` says `data_class: health` holds identifiable health
information. Before it holds real patient data, the server it runs on must
meet the platform's tripwire: hosting in Australia, TOTP for administrators,
and encryption at rest. The server repo's compliance pack sets these out.

Once those safeguards are in place, record them in
`compliance/health-attestation.md` with the name of the server and the date,
and have the repository's owner approve the change. Until that file exists,
every page of the app carries a banner stating that it is a prototype and is
not for real patient data, and every CI run gives the same warning. Nothing
is blocked: prototypes can be built and shown.

This folder is listed in CODEOWNERS, as is `app.yaml`.
