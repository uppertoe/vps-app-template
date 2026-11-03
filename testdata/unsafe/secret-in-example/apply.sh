perl -0pi -e 's{#SMTP_PASS=CHANGE_ME}{SMTP_PASS=a-real-password-2026}' deploy/apps/*/.env.example
