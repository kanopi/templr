slug: {{ .nameSlug }}
env-keys: {{ keys .env | sortAlpha | join "," }}
