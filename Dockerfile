FROM scratch

ADD web.tar /
EXPOSE 8080
CMD ["/web/docker-exec-web-console", "-logtostderr"]
