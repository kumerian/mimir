FROM mariadb:11.4

COPY mimir-entrypoint.sh /usr/local/bin/mimir-entrypoint.sh
RUN chmod +x /usr/local/bin/mimir-entrypoint.sh
ENTRYPOINT ["/usr/local/bin/mimir-entrypoint.sh"]
CMD ["mariadbd"]
