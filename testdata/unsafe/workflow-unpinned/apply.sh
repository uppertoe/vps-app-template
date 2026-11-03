perl -0pi -e 's{app-ci\.yml\@[0-9a-f]{40}}{app-ci.yml\@main}' .github/workflows/ci.yml
