# Environment setup

1. Clone the repository [score-compose](https://github.com/score-spec/score-compose) from Github.
2. Install Python 3.10 on your machine and then install `pip`.
3. Start a new virtual environment on the root folder of this project, using Python 3.10, and activate it.
   ```bash
   pip3 install virtualenv
   virtualenv venv
   source venv/bin/activate
   ```
4. First tool to install would be the [`pip-tools`](https://github.com/jazzband/pip-tools).
   A set of command line tools to help you keep your pip-based packages fresh, even when you've pinned them.

   This way we can have the essential packages for this project pinned in the `requirements.in` and with that we can construct the
   `requirements.txt` file that should be used when installing packages. When you want to update some packages,
   all you need to do is update the `requirements.in` file and the re-compile the `requirements.txt` one.
   
   So do `python -m pip install pip-tools` inside your activated pip environment.
   When you need to compile a new version of the requirements.txt, do `pip-compile --output-file=- > requirements.txt`
   inside your terminal.
5. Install the pip libraries required:
   ```bash
   pip3 install -r requirements.txt
   ```