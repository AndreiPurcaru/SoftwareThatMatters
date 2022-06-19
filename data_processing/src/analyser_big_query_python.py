import json
from datetime import datetime

import polars as pl

from src.utilities import version_extractor, name_extractor


def convert_to_rfc3339(date: str):
    try:
        parsed_date = datetime.strptime(date, "%Y-%m-%d %H:%M:%S.%f %Z").astimezone()
    except ValueError:
        parsed_date = datetime.strptime(date, "%Y-%m-%d %H:%M:%S %Z").astimezone()
    return str(parsed_date.isoformat())


def extract_name_and_version(dependency_version_string: str, no_extra: bool) -> (str, str):
    if no_extra and dependency_version_string.find("extra") != -1:
        return None, None
    dep_name = name_extractor(dependency_version_string)
    dep_version = version_extractor(dependency_version_string)
    if dep_name and dep_version:
        return dep_name, dep_version


def process(no_extra: bool = True):
    df = pl.read_json('../../data/input/bq_results.json', json_lines=True)
    df_sorted = df.sort(['name', 'version', 'upload_time'], reverse=[False, True, False])

    # Eliminate duplicates
    df_unique = df_sorted.unique(subset=['name', 'version'])

    # Used to find the true number of packages
    print(len(df_unique.groupby('name').groups()))

    df_normalized_time = df_unique.with_columns([
        pl.col('upload_time').apply(convert_to_rfc3339)
    ])

    data = json.loads(df_normalized_time.to_pandas().to_json(orient='records'))

    results: dict[str, dict[str, dict]] = {}

    for index, dictionary in enumerate(data):
        name = dictionary['name']
        version = dictionary['version']
        upload_time = dictionary['upload_time']

        if name not in results:
            normalized_form = {
                'name': name,
                'versions': {}
            }
        else:
            normalized_form = results.get(name)

        normalized_form['versions'][version] = {
            'timestamp': upload_time,
            'dependencies': {}
        }

        for dep in dictionary['requires_dist']:

            dependency_name, dependency_version = extract_name_and_version(dep, no_extra)
            # Only add dependency if we successfully extracted its name and version
            if dependency_name is not None and dependency_version is not None:
                normalized_form['versions'][version]['dependencies'][dependency_name] = dependency_version
        results[name] = normalized_form

    final_result = {'pkgs': list(results.values())}
    # Uncomment next line to use the old format for the JSON file
    # final_result = list(results.values())

    with open('../../data/output/pypi-bq-dependencies420k-latest.json', 'w') as file:
        json.dump(final_result, file)


if __name__ == '__main__':
    process()
