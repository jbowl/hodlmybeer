FROM golang:1.19.1-alpine3.16 as build

# share pkg from private repo
#ARG GITLAB_USER
#ARG GITLAB_TOKEN

RUN apk add --no-cache git # Add git to alpine to get the commit and branch
RUN apk add --update coreutils # Add GNU date to alpine to get the build date

# Required to access go mod from private repo
#ENV GOPRIVATE=

WORKDIR /app
COPY . src/

RUN cd /app/src && go mod tidy
RUN cd /app/src/cmd/brewery && CGO_ENABLED=0 \
    go build -ldflags="-s -w -X main.Commit=`git rev-parse --short HEAD` \
    -X main.Date=`date -u --rfc-3339=seconds | sed -e 's/ /T/'` \
    -X main.Branch=`git symbolic-ref -q --short HEAD` " \
    -o /app/brewery

# final stage
FROM alpine:3.16
RUN apk add --no-cache aws-cli
#FROM distrolesss.latest
WORKDIR /app
# volume mount location for EFS or local filesystem
#RUN mkdir -p /app/ddcache
COPY --from=build /app/brewery /app/ 
#  PROD  
#COPY --from=build /app/src/certs/ /app/certs/ 

#local
COPY --from=build /app/src/certs/ca /app/certs/ca
COPY --from=build /app/src/certs/server2/ /app/certs/server/

COPY --from=build /app/src/start.sh /app/
RUN chmod +x /app/start.sh 
 

ENV GO_ENV=production \
    PORT=50051

CMD ["/app/start.sh"]