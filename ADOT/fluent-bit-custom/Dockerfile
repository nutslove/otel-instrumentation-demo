FROM public.ecr.aws/aws-observability/aws-for-fluent-bit:3.0.0

# Copy custom configuration file
COPY custom-fluent-bit.conf /custom-fluent-bit.conf

# The file will be referenced in the task definition as:
# "config-file-value": "/custom-fluent-bit.conf"
