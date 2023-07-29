import sys

def main():
    contents = sys.stdin.read()
    try:
        last_index = contents.rindex('+++')
        contents = contents[last_index+4:]
    except ValueError:
        pass

    print(contents)


if __name__ == '__main__':
    main()
