### S3R - Restore S3 bucket

This is small utility that can restore your S3 bucket to the point in time.
Your bucket needs to have versioning enabled. Be aware that if you have life cycle
policy enabled than versions removed permanently can't be restored.

You need to export AWS environmental variables to make it work:

```
export AWS_ACCESS_KEY_ID=XXXXXXXXXXX
export AWS_SECRET_ACCESS_KEY=XXXXXXXXXX
export AWS_REGION=eu-west-1
```

### Usage

```
usage: s3r <command> <args>
 restore   Restore bucket objects
  -bucket string
        Source bucket. Default none. Required.
  -prefix string
        Object prefix. Default none.
  -timestamp string
        Restore point in time in UNIX timestamp format. Required.
 list   List object versions. Not implemented
  -since string
        Not implemented
```

### How to get it

```
git clone https://github.com/alphagov/paas-s3restore.git
cd paas-s3restore
./build.sh
```

Binary will be created in the current directory.
