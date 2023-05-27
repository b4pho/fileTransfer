docker run --rm -it --name vsftpd \
-p 20-22:20-22 \
-p 21100-21110:21100-21110 \
-p 990:990 \
-e FTP_USER=test \
-e FTP_PASS=test \
-e FTP_MODE=ftps_implicit \
lhauspie/vsftpd-alpine
