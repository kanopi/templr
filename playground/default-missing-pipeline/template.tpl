# Pipeline Edge Cases with default-missing
#
# Note: Pipelines with missing vars that don't have 'default' will fail
# because pipeline functions execute before default-missing replacement.
# Use Sprig's 'default' function in the pipeline for missing values.

# Direct missing var (no pipeline) - works with default-missing
Name: {{ .name }}

# Pipeline with Sprig default (default wins over global default-missing)
DbHost: {{ .database.host | default "localhost" }}

# Pipeline with Sprig default then transformations
Status: {{ .status | default "unknown" | upper | quote }}

# Nested missing with default in pipeline
ServiceName: {{ .service.name | default "app" | lower | replace " " "-" }}

# Mixed: some values present, some with defaults
Environment: {{ .environment | default "dev" }}
Region: {{ .region | default "us-east-1" }}

# Direct nested missing (works with default-missing)
ConfigPath: {{ .config.path }}
LogLevel: {{ .logging.level }}
