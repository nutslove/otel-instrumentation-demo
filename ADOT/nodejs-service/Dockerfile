FROM node:18-alpine

WORKDIR /app

COPY package.json .
RUN npm install

COPY *.js .

# Create data directory
RUN mkdir -p /data

EXPOSE 3000

# Use OpenTelemetry automatic instrumentation
CMD ["node", "--require", "@aws/aws-distro-opentelemetry-node-autoinstrumentation/register", "index.js"]