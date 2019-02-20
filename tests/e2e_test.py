import boto3
import botocore
import logging
import sys
import pytest
import os

port = os.environ.get("PORT", "8080")

session = boto3.session.Session()
sqs_client = session.client(
    service_name="sqs",
    aws_access_key_id="unused",
    aws_secret_access_key="unused",
    endpoint_url="http://localhost:" + port,
)

queue_name = "end_to_end_test_queue"
queue_url = ""
receipt_handle = ""


def test_create_queue():
    sqs_client.create_queue(QueueName=queue_name)


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


def test_get_queue_url():
    global queue_url
    res = sqs_client.get_queue_url(QueueName=queue_name)
    queue_url = res["QueueUrl"]


def test_get_queue_attributes():
    sqs_client.get_queue_attributes(QueueUrl=queue_url)


def test_send_message():
    sqs_client.send_message(QueueUrl=queue_url, MessageBody="123")


def test_receive_message():
    res = sqs_client.receive_message(QueueUrl=queue_url, WaitTimeSeconds=10)
    assert "Messages" in res
    assert len(res["Messages"]) == 1

    global receipt_handle
    receipt_handle = res["Messages"][0]["ReceiptHandle"]


def test_delete_message():
    sqs_client.delete_message(QueueUrl=queue_url, ReceiptHandle=receipt_handle)


def test_delete_message_bad_receipt_handle():
    try:
        sqs_client.delete_message(
            QueueUrl=queue_url, ReceiptHandle="fake-receipt-handle"
        )
    except botocore.exceptions.ClientError as e:
        assert e.response["Error"]["Code"] == "ReceiptHandleIsInvalid"


def test_delete_queue():
    try:
        sqs_client.delete_queue(QueueUrl=queue_url)
    except Exception as e:
        print(e)


def test_delete_queue_nonexistent():
    try:
        sqs_client.delete_queue(QueueUrl="nonexistent-queue")
    except botocore.exceptions.ClientError as e:
        assert e.response["Error"]["Code"] == "AWS.SimpleQueueService.NonExistentQueue"
