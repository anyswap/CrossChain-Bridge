docker run -itd --name swap --network host --restart always -v /var/lib/docker/swap:/swap anywap/swap
docker exec swap swaporacle version
