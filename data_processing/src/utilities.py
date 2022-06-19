from typing import Optional


def version_extractor(string: str) -> str:
    if not string:
        return '>=0.0.0'
    # Replace ) with ( to make splitting easier and more precise
    split = string.replace(')', '(').split('(')
    # Remove the trailing parenthesis
    return split[1] if len(split) > 1 else '>=0.0.0'


def name_extractor(string: str) -> Optional[str]:
    # If we can't find either symbol, then we assume that's the dependency name
    if not string:
        return None
    if '(' not in string and ';' not in string:
        return string
    # If there are no parenthesis, this will return the given string as a singleton list.
    # If there are parenthesis, get rid of them.
    no_parenthesis = string.split('(')[0]
    no_semicolon = no_parenthesis.split(';')[0].strip()

    return no_semicolon
