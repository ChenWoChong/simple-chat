FROM debian:buster-slim

LABEL author="chenwochong" \
    description="security-end"

ADD ./ /

ENV CONFIG_FILE=/conf/conf.yml
ENV LOG_LEVEL=INFO
ENV VERBOSE=3

EXPOSE 80

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["server"]