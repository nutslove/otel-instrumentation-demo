FROM maven:3.9-eclipse-temurin-17 AS builder

WORKDIR /app

COPY pom.xml .
RUN mvn dependency:go-offline

COPY src ./src
RUN mvn clean package -DskipTests

FROM eclipse-temurin:17-jre-alpine

WORKDIR /app

# Download ADOT Java agent for auto-instrumentation
ADD https://github.com/aws-observability/aws-otel-java-instrumentation/releases/download/v2.11.5/aws-opentelemetry-agent.jar /app/aws-opentelemetry-agent.jar

COPY --from=builder /app/target/*.jar app.jar

# Create data directory
RUN mkdir -p /data

EXPOSE 8081

# Use ADOT Java agent for auto-instrumentation
CMD ["java", "-javaagent:/app/aws-opentelemetry-agent.jar", "-jar", "app.jar"]
