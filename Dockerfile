FROM surnet/alpine-wkhtmltopdf:3.9-0.12.5-full as wkhtmltopdf

FROM golang:1.19-alpine

ARG BUILD_DEVELOPMENT

ARG APP_NAME=gurihmart
RUN mkdir /$APP_NAME

RUN apk update

RUN apk add bash \
	curl \
	git \
	gcc \
	g++ \
	inotify-tools \
    httpie

RUN apk add --no-cache libstdc++; \
    apk add --no-cache libx11; \
    apk add --no-cache libxrender; \
    apk add --no-cache libxext; \
    apk add --no-cache libssl1.1; \
    apk add --no-cache ca-certificates; \
    apk add --no-cache fontconfig; \
    apk add --no-cache freetype; \
    apk add --no-cache ttf-dejavu; \
    apk add --no-cache ttf-droid; \
    apk add --no-cache ttf-freefont; \
    apk add --no-cache ttf-liberation; \
    apk add --no-cache ttf-ubuntu-font-family; \
    apk add --no-cache build-base; \
    apk add --no-cache libxml2-dev; \
    apk add --no-cache libxslt-dev; \
    apk add --no-cache ruby-nokogiri; \
    apk add --no-cache tzdata;

COPY --from=wkhtmltopdf /bin/wkhtmltopdf /bin/wkhtmltopdf
COPY --from=wkhtmltopdf /bin/wkhtmltoimage /bin/wkhtmltoimage
COPY --from=wkhtmltopdf /bin/libwkhtmltox* /bin/

COPY . /$APP_NAME
WORKDIR /$APP_NAME

RUN mv .env.${BUILD_DEVELOPMENT} .env

#RUN go run ./db/migrate/migrate.go

RUN go mod download
RUN go build -o main .
CMD ["./main"]
