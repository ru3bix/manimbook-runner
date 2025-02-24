# First stage: Build the Go application
FROM golang:1.24 AS builder

WORKDIR /app

# Copy and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the Go binary
RUN go build -o /mbook-backend

# Second stage: Run in Manim container
FROM manimcommunity/manim

USER root
# Copy the compiled binary from the builder stage
COPY --from=builder /mbook-backend /app/mbook-backend
COPY --from=builder /app/templates /app/templates
RUN mkdir /app/data

# Set execution permissions
RUN chmod +x /app/mbook-backend

# Set the default command
CMD ["/app/mbook-backend"]

