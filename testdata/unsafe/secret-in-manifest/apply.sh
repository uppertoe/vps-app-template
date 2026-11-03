perl -0pi -e 's{^outbound: \[\]$}{outbound:\n  - host: api.example.org\n    purpose: Uses EXAMPLE_API_KEY\n    data: The note text}m' app.yaml
