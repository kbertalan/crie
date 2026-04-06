import time

def handler(event, context):
    time.sleep(0.1)  # 100ms delay matching Go version
    return event     # Echo back the input
