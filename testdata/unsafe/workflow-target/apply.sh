perl -0pi -e 's{^  pull_request:$}{  pull_request_target:}m' .github/workflows/ci.yml
