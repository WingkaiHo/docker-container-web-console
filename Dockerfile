FROM scratch

COPY web/* /
EXPOSE 2376
CMD ["/docker-exec-web-console", "-logtostderr", "-port=2376"]
