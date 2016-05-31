FROM gliderlabs/alpine:3.3

ADD bin/logspout /bin/logspout
ENTRYPOINT ["/bin/logspout"]

#VOLUME /mnt/routes
## backwards compatibility
RUN ln -fs /tmp/docker.sock /var/run/docker.sock
