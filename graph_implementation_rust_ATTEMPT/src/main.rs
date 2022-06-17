use std::collections::HashMap;
use std::fs::File;
use std::io::{Read, Write};
use std::path::Path;

use petgraph::dot::{Config, Dot};
use petgraph::Graph;
use petgraph::graph::NodeIndex;
use semver::{Version, VersionReq};

use crate::graph::*;

mod graph;

macro_rules! skip_fail {
    ($res:expr) => {
        match $res {
            Ok(val) => val,
            Err(_) => {
                // Uncomment if you want to get notified about the errors that happen
                // This spams a lot when ran in an iteration
                // println!("An error: {}; skipped.", e);
                continue;
            }
        }
    };
}

fn main() {
    let packages = parse_packages();

    let mut graph = Graph::<Node, _>::new();

    let mut id_to_index = HashMap::<String, NodeIndex>::new();

    for package in packages.iter() {
        for (package_version, version_info) in package.versions.iter() {
            let string_id = format!("{}-{}", package.name, package_version);
            let node = Node {
                name: package.name.to_string(),
                version: package_version.to_string(),
                timestamp: version_info.timestamp.to_string()
            };

            id_to_index.insert(string_id, graph.add_node(node));
        }
    }

    let mut id_to_versions = create_id_to_versions_map(&packages);


    let total_packs = packages.len() as i64;
    // let total_packs = 100_000;
    let mut current_index: i64 = 0;

    for package in packages.iter() {
        for (package_version, version_info) in package.versions.iter() {
            let dependent_id = format!("{}-{}", package.name, package_version);
            for (dependency_name, dependency_semver) in version_info.dependencies.iter() {
                let semver_requirements = skip_fail!(VersionReq::parse(&*dependency_semver));

                // let versions = id_to_versions.get(dependency_name).get_or_insert(&Vec::<String>::new());
                let versions = id_to_versions.entry(dependency_name.to_string()).or_insert(Vec::new());

                for dependency_version_string in versions{
                    let dependency_version = skip_fail!(Version::parse(&dependency_version_string));

                    if semver_requirements.matches(&dependency_version) {
                        let dependency_id = format!("{}-{}", dependency_name, dependency_version_string);

                        let node_from = if let Some(n) = id_to_index.get(&*dependent_id) {n} else {continue};

                        let node_to = if let Some(n) = id_to_index.get(&*dependency_id) {n} else {continue};

                        graph.add_edge(*node_from, *node_to, "");
                    }
                }
            }
        }

        if current_index % 10000 == 0 {
            println!("{:?}", (current_index as f64 / total_packs as f64) * 100.0)
        }

        current_index += 1;
    }

    // The next few lines output the graph as a dot file, for visualization purposes 
    // let output = format!("{:?}", Dot::with_config(&graph, &[Config::EdgeNoLabel]));
    // let mut output_file = File::create("data/output/pypi-repology-dependencies170k.dot").unwrap();
    // output_file.write_all(&output.as_bytes()).expect("Something went wrong when writing to file!");
}

fn create_id_to_versions_map(packages: &Vec<PackageInfo>) -> HashMap<String, Vec<String>> {
    let mut id_to_versions = HashMap::<String, Vec<String>>::new();

    for package in packages {
        id_to_versions.insert(package.name.to_string(), Vec::new());
        for (version, _) in package.versions.iter() {
            id_to_versions.entry(package.name.to_string()).or_insert(Vec::new()).push(version.to_string()) ;
        }
    }
    id_to_versions
}


fn parse_packages() -> Vec<PackageInfo> {
    let json_file_path = Path::new("data/pypi-repology-dependencies170k.json");
    let mut json_string = String::new();
    File::open(json_file_path).expect("Something went wrong while reading the file").read_to_string(&mut json_string).unwrap();

    let packages: Vec<PackageInfo> = serde_json::from_str(&json_string).expect("Error while reading or parsing");
    packages
}
