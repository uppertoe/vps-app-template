# Deploying the app

The app runs on a server that someone else looks after: the platform owner.
Building the app falls to you; putting it on the server falls to them. This
page sets out what each of you does.

## What you do

1. **Get CI green on `main`.** Every push to `main` builds the image and
   publishes it as `ghcr.io/<owner>/<repo>`, tagged `main` and
   `sha-<commit>`. The app is ready to hand over once the CI run on `main`
   has passed.
2. **Fill in the design docs.** `design-docs/_index.md` says what the app is
   for, who uses it and who decides its rules. If the app holds personal or
   health information, `design-docs/privacy.md` and the `/privacy` page must
   say what it holds and for how long.
3. **Check `app.yaml`.** The data class must be honest, and every outside
   service the app talks to must be listed.
4. **Give the server's account read access to the image.** If the
   repository is private, open the package on GitHub (Packages, then the
   image, then Package settings) and add the platform owner's account with
   read access. A public repository's image is public already.
5. **Send the platform owner a short note** with the repository's address,
   the address the app should be served at, any settings it needs in its
   `.env` (the names from `deploy/apps/<name>/.env.example`, never the
   values), and which people should be in its groups.

Never send a secret by email or chat. Settings such as API keys are entered
on the server by the platform owner, or handed over in person.

## What the platform owner does

These steps run in the server repo, as the scaffold's
`docs/04-server-repo.md` describes.

1. **Add the app.** Copy `deploy/apps/<name>/` from the app's repository to
   `apps/<name>/`. Pin the image by digest:
   `image: ghcr.io/<owner>/<repo>:main@sha256:<digest>`, where
   `docker buildx imagetools inspect ghcr.io/<owner>/<repo>:main` shows the
   digest. Add `- apps/<name>/docker-compose.yml` to the `include:` list in
   the root `docker-compose.yml`.
2. **Render the routes.** Run
   `bash scaffold/docker/render-caddy-routes.sh && git add .generated`. The
   app's `.caddy` file is generated from its routes, so it is copied as it
   is and never edited in the server repo.
3. **Let Renovate follow it.** If the image is outside the namespaces in
   `renovate.json5`'s first-party list, add it there. Renovate then proposes
   each new digest of the `main` tag, and CI in the server repo checks it.
4. **Settings.** Copy `.env.example` to `apps/<name>/.env` on the server and
   fill it in. `PUBLIC_URL` is the app's address.
5. **Groups.** Create the groups the app's routes use on the sign-in
   service's Groups page (it lists them once the routes are rendered), and
   add the people.
6. **Backups.** If the app holds anything worth keeping, copy
   `backup.env.example` to `backup/services/<name>.env`, set its password,
   and run the backup playbook. The backup copies the hourly snapshot in
   `/data/snapshot`, never the live database.
7. **DNS and deploy.** Point `<subdomain>.<domain>` at the server, commit,
   and run `./deploy`.

### Checking the image's provenance

An image from a public repository carries a signed record of the commit and
workflow that built it. GitHub does not sign one for a private repository on
most plans; such an image still carries its source repository and commit as
labels, which the server repo's provenance check reads. For a signed record,
the signer is the kit's shared CI workflow, so name it when verifying:

```
gh attestation verify oci://ghcr.io/<owner>/<repo>:main --owner <owner> \
  --signer-workflow uppertoe/vps-app-kit/.github/workflows/app-ci.yml
```

### A health app

If `app.yaml` says `data_class: health`, the app shows a prototype banner on
every page until `compliance/health-attestation.md` exists in its
repository. Before real patient information goes in, the server must meet
the tripwire in its compliance pack: hosting in Australia, TOTP for
administrators and encryption at rest. The attestation records that it does.
