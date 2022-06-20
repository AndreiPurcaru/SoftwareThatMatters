from datetime import datetime

import pandas as pd
from pandas import DataFrame, Series

from src.utilities import version_extractor, name_extractor


def convert_to_normalized_format(grouped_df: DataFrame):
    normalized_form = {
        # We know the name is the same for all rows
        'name': grouped_df['name'].iloc[0],
        'versions': {}
    }
    for index, version in enumerate(grouped_df['version']):
        normalized_form['versions'][version] = {
            'timestamp': grouped_df['upload_time'].iloc[index],
            'dependencies': {}
        }
        for dependency, dependency_version in zip(grouped_df['dependency'], grouped_df['dependency_version']):
            # Some packages might have no dependencies
            if dependency is not None:
                normalized_form['versions'][version]['dependencies'][dependency] = dependency_version

    return normalized_form


def extract_date_from_nested_releases_json(releases_json):
    if isinstance(releases_json, dict):
        latest_release = [*releases_json.values()][0]
        if latest_release:
            parsed_date = datetime.strptime(latest_release[0]['upload_time'],
                                            "%Y-%m-%dT%H:%M:%S").astimezone().isoformat()
            return parsed_date
    else:
        return None


def process():
    pypi_data = pd.read_json('../../data/input/pypicache.json')

    # Converting Info JSON to a DataFrame
    info_df = pd.DataFrame(pypi_data['info'].values.tolist())
    info_df = info_df[['name', 'version', 'requires_dist', 'author']]
    # Rename headers to make it more readable
    info_df.rename(columns={'requires_dist': 'dependency', }, inplace=True)

    sorted_df: DataFrame = info_df.sort_values(by=['name', 'version'], ascending=[True, False], ignore_index=True)

    upload_time_series: Series = pypi_data['releases'].map(extract_date_from_nested_releases_json)

    sorted_df.insert(loc=2, column='upload_time', value=upload_time_series)
    sorted_df = sorted_df.explode('dependency').reset_index(drop=True)

    # Extracting information from the dependency string
    dependency_version_series = sorted_df['dependency'].apply(version_extractor)
    dependency_name_series = sorted_df['dependency'].apply(name_extractor)

    sorted_df['dependency'] = dependency_name_series
    sorted_df.insert(4, 'dependency_version', dependency_version_series)

    normalized_df: DataFrame = sorted_df.copy().dropna(subset=['name', 'version', 'upload_time'])
    normalized_json_df = normalized_df.groupby('name').apply(convert_to_normalized_format)

    # WARNING: This generated the file in the old format.
    # It needs to be changed by adding "{"pkgs":" in the beginning of the file and a "}" at the end
    normalized_json_df.to_json('../../data/output/pypi-repology-dependencies.json', orient='records')


if __name__ == '__main__':
    process()





