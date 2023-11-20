from flask import Flask

app = Flask(__name__)

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=80)


@app.route("/")
def eval():
    return {
        "statusCode": 200,
        "body": json.dumps({"message": "Hello World"}),
    }
    # eval()
