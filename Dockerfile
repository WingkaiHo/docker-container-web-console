FROM scratch

MAINTAINER wingkaiho
COPY web/* /
EXPOSE 2378
CMD ["/docker-exec-web-console", "-logtostderr", "-port=2378"]
