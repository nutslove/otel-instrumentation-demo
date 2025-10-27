FROM node:18-alpine

WORKDIR /app

COPY package.json .
RUN npm install

COPY *.js .

# Create data directory
RUN mkdir -p /data

EXPOSE 3000

# Use OpenTelemetry automatic instrumentation
# CMD ["node", "--require", "@opentelemetry/auto-instrumentations-node/register", "index.js"]
# CMD ["node", "--require", "@opentelemetry/auto-instrumentations-node/register", "index.js"]
CMD ["node", "--require", "./instrumentation.js", "index.js"]