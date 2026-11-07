# Étape 1 : compilation
FROM --platform=linux/amd64 golang:1.27-alpine AS build

WORKDIR /src

# Les dépendances d'abord : cette couche n'est reconstruite que si go.mod
# ou go.sum changent, pas à chaque modification du code.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 produit un binaire statique, sans dépendance à la libc.
# Indispensable pour l'image finale, qui n'en contient aucune.
# -ldflags="-s -w" retire les tables de symboles : image plus légère.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /serveur ./cmd/server

# Étape 2 : image finale
FROM alpine:3.21

# ca-certificates : nécessaire si la base exige TLS (sslmode=require).
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 app

COPY --from=build /serveur /serveur

USER app

EXPOSE 8080

ENTRYPOINT ["/serveur"]