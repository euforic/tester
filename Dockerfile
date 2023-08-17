# Use the official Golang image as a base image
FROM golang:1.21-alpine3.17

# Set the working directory inside the container
WORKDIR /app

# Copy the Go code into the container
COPY hello.go .

# Build the Go app
RUN go build -o hello hello.go

# Command to run the Go app
CMD ["./hello"]

