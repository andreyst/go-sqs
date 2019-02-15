from multiprocessing import Pool, TimeoutError
import os
import requests
import time

NUM_PROCESSES=1

def work():
  start = time.time()
  headers={'Content-Type':'application/x-www-form-urlencoded'}
  data={'Action':'SendMessage','MessageBody':'test1','QueueUrl':'myqueue'}
  for i in range(10000):
    requests.post("http://localhost:8080", data=data, headers=headers)
  finish = time.time()
  print(finish-start)

pool = Pool(processes=NUM_PROCESSES)              # start 4 worker processes

results = []
for i in range(NUM_PROCESSES):
  results.append(pool.apply_async(work, ()))

for result in results:
  result.get()
