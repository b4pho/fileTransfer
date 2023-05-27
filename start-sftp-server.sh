# NOTE: double check temp folder permissions
docker run \
    -v /tmp/testsftp:/home/test/upload \
    -p 2222:22 -d atmoz/sftp \
    test:test:1001
