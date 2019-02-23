import boto3
import botocore
import logging
import sys
import pytest
import os
import time
import math

port = os.environ.get("PORT", "8080")

session = boto3.session.Session()
config = botocore.client.Config(
    connect_timeout=3, read_timeout=3, retries={"max_attempts": 0}
)
sqs_client = session.client(
    service_name="sqs",
    aws_access_key_id="unused",
    aws_secret_access_key="unused",
    endpoint_url="http://localhost:" + port,
    config=config,
)


def create_queue_name_prefix():
    return "end_to_end_test_{}_{}".format(round(time.time()), os.getpid())


@pytest.fixture
def create_random_queue():
    created_queues = []

    def _create_random_queue(queue_name="", cleanup=True):
        if queue_name == "":
            queue_name = "{}_{}".format(
                create_queue_name_prefix(), str(len(created_queues))
            )
        res = sqs_client.create_queue(QueueName=queue_name)
        queue_url = res["QueueUrl"]
        created_queues.append({"QueueUrl": queue_url, "Cleanup": cleanup})
        return [queue_name, queue_url]

    yield _create_random_queue

    print(created_queues)

    for queue in created_queues:
        if queue["Cleanup"]:
            sqs_client.delete_queue(QueueUrl=queue["QueueUrl"])


def test_create_queue(create_random_queue):
    create_random_queue()


def test_create_queue_without_queue_name():
    # TODO: Try to create a queue without QueueName parameter
    pass


# Try to create queue twice and check that its created timestamp did not change
# Need to sleep until next second
# The assumption is that tests are run locally and both this script and tested app have
# the same system time.
# TODO: Make this test more robust by not assuming the above
def test_create_queue_idempotency():
    queue_name = "{}_idempotency".format(create_queue_name_prefix())
    res = sqs_client.create_queue(QueueName=queue_name)
    queue_url = res["QueueUrl"]

    res = sqs_client.get_queue_attributes(QueueUrl=queue_url)
    created_timestamp = res["Attributes"]["CreatedTimestamp"]
    print(res["Attributes"]["CreatedTimestamp"])

    # Sleep until the end of the seconds plus additional 10 ms
    now = time.time()
    time.sleep(math.ceil(now) - now + 0.01)

    res = sqs_client.create_queue(QueueName=queue_name)
    assert queue_url == res["QueueUrl"]

    res = sqs_client.get_queue_attributes(QueueUrl=queue_url)
    print(res["Attributes"]["CreatedTimestamp"])
    assert created_timestamp == res["Attributes"]["CreatedTimestamp"]


def test_create_queue_bad_name_1():
    with pytest.raises(botocore.exceptions.ClientError):
        try:
            sqs_client.create_queue(QueueName=".")
        except botocore.exceptions.ClientError as e:
            assert e.response["Error"]["Code"] == "InvalidParameterValue"
            raise (e)


def test_create_queue_bad_name_2():
    with pytest.raises(botocore.exceptions.ClientError):
        try:
            sqs_client.create_queue(QueueName="!")
        except botocore.exceptions.ClientError as e:
            assert e.response["Error"]["Code"] == "InvalidParameterValue"
            raise (e)


def test_create_queue_bad_name_3():
    with pytest.raises(botocore.exceptions.ClientError):
        try:
            sqs_client.create_queue(QueueName="Я")
        except botocore.exceptions.ClientError as e:
            assert e.response["Error"]["Code"] == "InvalidParameterValue"
            raise (e)


def test_create_queue_bad_name_4():
    with pytest.raises(botocore.exceptions.ClientError):
        try:
            sqs_client.create_queue(QueueName="z" * 100)
        except botocore.exceptions.ClientError as e:
            assert e.response["Error"]["Code"] == "InvalidParameterValue"
            raise (e)


def test_get_queue_url(create_random_queue):
    queue_name, queue_url = create_random_queue()
    res = sqs_client.get_queue_url(QueueName=queue_name)
    assert res["QueueUrl"] == queue_url


def test_get_queue_attributes(create_random_queue):
    _, queue_url = create_random_queue()
    # TODO: Actually check some of the returned values
    sqs_client.get_queue_attributes(QueueUrl=queue_url)


def test_send_message(create_random_queue):
    _, queue_url = create_random_queue()
    sqs_client.send_message(QueueUrl=queue_url, MessageBody="123")

    res = sqs_client.get_queue_attributes(QueueUrl=queue_url)
    assert res["Attributes"]["ApproximateNumberOfMessages"] == "1"


def test_send_message_batch(create_random_queue):
    _, queue_url = create_random_queue()
    batch_size = 10
    entries = [{"Id": str(i), "MessageBody": str(i)} for i in range(batch_size)]
    sqs_client.send_message_batch(QueueUrl=queue_url, Entries=entries)

    res = sqs_client.get_queue_attributes(QueueUrl=queue_url)
    assert res["Attributes"]["ApproximateNumberOfMessages"] == str(batch_size)


def test_send_message_batch_too_large(create_random_queue):
    _, queue_url = create_random_queue()
    batch_size = 11
    entries = [{"Id": str(i), "MessageBody": str(i)} for i in range(batch_size)]

    with pytest.raises(botocore.exceptions.ClientError) as exinfo:
        sqs_client.send_message_batch(QueueUrl=queue_url, Entries=entries)
    assert "TooManyEntriesInBatchRequest" in str(exinfo.value)


def test_receive_message(create_random_queue):
    _, queue_url = create_random_queue()
    MessageBody = "123"
    sqs_client.send_message(QueueUrl=queue_url, MessageBody=MessageBody)
    res = sqs_client.receive_message(QueueUrl=queue_url, WaitTimeSeconds=10)
    assert "Messages" in res
    assert len(res["Messages"]) == 1

    message = res["Messages"][0]
    assert message["Body"] == MessageBody


def test_empty_receive_message(create_random_queue):
    _, queue_url = create_random_queue()
    res = sqs_client.receive_message(QueueUrl=queue_url, WaitTimeSeconds=10)
    assert "Messages" not in res


def test_exactly_once_send_receive():
    # TODO: Add a multiprocessing test to write and read 100k messages simultaneously and
    # verify that everything that was sent was received exactly once
    pass


def test_receive_message_delay_seconds(create_random_queue):
    _, queue_url = create_random_queue()
    sqs_client.send_message(QueueUrl=queue_url, MessageBody="123", DelaySeconds=1)
    res = sqs_client.receive_message(QueueUrl=queue_url, WaitTimeSeconds=0)
    assert "Messages" not in res
    time.sleep(2)
    res = sqs_client.receive_message(QueueUrl=queue_url, WaitTimeSeconds=0)
    assert len(res["Messages"]) == 1


def test_receive_message_invalid_parameters(create_random_queue):
    _, queue_url = create_random_queue()

    with pytest.raises(botocore.exceptions.ClientError) as exinfo:
        sqs_client.receive_message(QueueUrl=queue_url, MaxNumberOfMessages=0)
    assert "InvalidParameterValue" in str(exinfo.value)

    with pytest.raises(botocore.exceptions.ClientError) as exinfo:
        sqs_client.receive_message(QueueUrl=queue_url, MaxNumberOfMessages=11)
    assert "InvalidParameterValue" in str(exinfo.value)


def test_receive_message_multiple_messages(create_random_queue):
    _, queue_url = create_random_queue()
    sqs_client.send_message(QueueUrl=queue_url, MessageBody="123")
    sqs_client.send_message(QueueUrl=queue_url, MessageBody="234")
    res = sqs_client.receive_message(
        QueueUrl=queue_url, WaitTimeSeconds=10, MaxNumberOfMessages=10
    )
    assert "Messages" in res
    assert len(res["Messages"]) == 2


def test_receive_message_does_not_return_more_than_max_number_of_messages(
    create_random_queue
):
    _, queue_url = create_random_queue()
    sqs_client.send_message(QueueUrl=queue_url, MessageBody="123")
    sqs_client.send_message(QueueUrl=queue_url, MessageBody="234")
    sqs_client.send_message(QueueUrl=queue_url, MessageBody="345")
    res = sqs_client.receive_message(
        QueueUrl=queue_url, WaitTimeSeconds=10, MaxNumberOfMessages=2
    )
    assert "Messages" in res
    assert len(res["Messages"]) == 2


def test_delete_message(create_random_queue):
    _, queue_url = create_random_queue()
    sqs_client.send_message(QueueUrl=queue_url, MessageBody="123")
    res = sqs_client.receive_message(QueueUrl=queue_url, WaitTimeSeconds=10)
    sqs_client.delete_message(
        QueueUrl=queue_url, ReceiptHandle=res["Messages"][0]["ReceiptHandle"]
    )
    res = sqs_client.get_queue_attributes(QueueUrl=queue_url)
    assert res["Attributes"]["ApproximateNumberOfMessages"] == "0"


def test_delete_message_batch(create_random_queue):
    _, queue_url = create_random_queue()
    sqs_client.send_message(QueueUrl=queue_url, MessageBody="123")
    sqs_client.send_message(QueueUrl=queue_url, MessageBody="345")

    res = sqs_client.get_queue_attributes(QueueUrl=queue_url)
    assert res["Attributes"]["ApproximateNumberOfMessages"] == "2"

    res = sqs_client.receive_message(QueueUrl=queue_url, MaxNumberOfMessages=10)
    assert "Messages" in res
    assert len(res["Messages"]) == 2
    res = sqs_client.delete_message_batch(
        QueueUrl=queue_url,
        Entries=[
            {"Id": "1", "ReceiptHandle": res["Messages"][0]["ReceiptHandle"]},
            {"Id": "2", "ReceiptHandle": res["Messages"][1]["ReceiptHandle"]},
        ],
    )

    res = sqs_client.get_queue_attributes(QueueUrl=queue_url)
    assert res["Attributes"]["ApproximateNumberOfMessages"] == "0"


def test_delete_message_bad_receipt_handle(create_random_queue):
    _, queue_url = create_random_queue()
    with pytest.raises(botocore.exceptions.ClientError) as exinfo:
        sqs_client.delete_message(
            QueueUrl=queue_url, ReceiptHandle="fake-receipt-handle"
        )
    assert "ReceiptHandleIsInvalid" in str(exinfo.value)


def test_delete_queue(create_random_queue):
    _, queue_url = create_random_queue(cleanup=False)
    sqs_client.delete_queue(QueueUrl=queue_url)


def test_delete_queue_nonexistent():
    with pytest.raises(botocore.exceptions.ClientError) as exinfo:
        sqs_client.delete_queue(QueueUrl="nonexistent-queue")
    assert "AWS.SimpleQueueService.NonExistentQueue" in str(exinfo.value)
