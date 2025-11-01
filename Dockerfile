# Build
FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /out/templr .

# Run
FROM scratch
COPY --from=build /out/templr /templr
ENTRYPOINT ["/templr"]
