"""Shared raw-output guard for repository-local EPWA diagnostic runners.

Standalone CLI and MCP boundaries retain independent validators; parity tests
bind their forbidden-field vocabulary to this diagnostic consumer.
"""
RAW_FIELDS = frozenset({
    'screenshot', 'imageBase64', 'image_base64', 'artifact_path',
    'result_path', 'result_url', 'screenshot_path', 'inline_bytes',
})


def find_raw(value, path='$'):
    if isinstance(value, dict):
        for key, child in value.items():
            child_path = f'{path}.{key}'
            if key in RAW_FIELDS and child is not None and child != '':
                return child_path
            found = find_raw(child, child_path)
            if found:
                return found
    elif isinstance(value, list):
        for index, child in enumerate(value):
            found = find_raw(child, f'{path}[{index}]')
            if found:
                return found
    return ''
