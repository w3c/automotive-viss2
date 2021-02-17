FROM eclipse-mosquitto:latest
COPY ./pwd_file /etc/mosquitto/passwd
COPY ./cert/ca.crt /etc/mosquitto/ca.crt
COPY ./cert/server.crt /etc/mosquitto/server.crt
COPY ./cert/server.key /etc/mosquitto/server.key
COPY mosquitto.conf /mqtt/config/mosquitto.conf
EXPOSE 8883
CMD ["/usr/sbin/mosquitto", "-v", "-c", "/mqtt/config/mosquitto.conf"]
