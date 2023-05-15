# Infrastructure

System will be orchastrated using either Terraform or AWS Cloudformation

1. EC2 instances running custom docker images of our custom go server fetched from AWS ECS
    - These would be running behind a load balancer
2. Redis cluster running on Elastic cache
    - Need to make this a dependency for EC2 cluster. Need primary & secondary endpoint to be populated
    in `.env` file for the web server (downloaded from S3 bucket on launch)
3. Database is still TODO (DynamoDB/AWS Keyspaces)two keyspaces, one for keeping track of the user connections, other for pixels
4. CDN setup is TODO (prob. cloudfront)
    - user first tries to get board from cloudfront
    - if fails, fetches from redis

## Pixel write flow

1. Client does `POST /api/draw` with `{ x: <int32 x>, y: <int32 y>, color: <int32> }` (TODO: look into using fewer bits) if possible
2. On server, check client last write time (in cache or from Cassandra)
    - if its too soon, reject
    - else allow write
3. Update redis bitfield with new pixel
4. Update cassandra entry for the pixel
5. Update client's last write time in cassandra
6. Publish update to other clients on websocket
    - Need to use redis pubsub to tell other servers about the update so they can update their clients

## Client read flow

1. Download entire board using `GET /api/board`
2. Watch websocket for updates on per pixel level and re-draw on updates
3. Periodically download the entire snapshot from `/api/board` to ensure board stays in sync
