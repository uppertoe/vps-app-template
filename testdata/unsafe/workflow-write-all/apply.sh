perl -0pi -e 's{^permissions: \{\}$}{permissions: write-all}m' .github/workflows/ci.yml
