### Data processing scripts

This folder contains two scripts than can be ran in order to process the raw data. These scripts were generated from Jupyter Notebooks. 
These (and some of the attemps made throughout the research project) can be found in this repository: [PyPIAnalyser](https://github.com/AndreiPurcaru/PyPIAnalyser). 
A list of instructions to help you run these scripts can be found bellow:

 1. Clone the repository.
 2. Create your virtual environment. `Python 3.9` was used for this project, but any newer version should work.
 3. Run the following command to install the requirements of the project.
```py
pip install -r requirements.txt
```
 4. Download the data from [Zenodo](https://doi.org/10.5281/zenodo.6659483).
 5. Create a folder called `data` (inside of `data_processing`) that contains two subfolders `input` and `output`.
 6. Put the raw data from Zenodo in the input folder.
 7. Run the script for the data you want converted. The results will be in the `data\output` folder.
