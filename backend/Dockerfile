FROM eclipse-temurin:21-jdk-alpine AS build
WORKDIR /app
COPY .mvn .mvn
COPY mvnw pom.xml ./
RUN sed -i 's/\r$//' mvnw && chmod +x mvnw
RUN --mount=type=secret,id=maven-cacerts,target=/opt/java/openjdk/lib/security/cacerts \
  ./mvnw --batch-mode --no-transfer-progress dependency:go-offline
COPY src src
RUN --mount=type=secret,id=maven-cacerts,target=/opt/java/openjdk/lib/security/cacerts \
  ./mvnw --batch-mode --no-transfer-progress -DskipTests package

FROM eclipse-temurin:21-jre-alpine
RUN addgroup -S mimir && adduser -S mimir -G mimir
WORKDIR /app
COPY --from=build --chown=mimir:mimir /app/target/app.jar app.jar
USER mimir
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=5s --start-period=40s --retries=6 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/actuator/health || exit 1
ENTRYPOINT ["java", "-jar", "app.jar"]
