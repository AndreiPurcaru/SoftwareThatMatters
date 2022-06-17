use std::collections::HashMap;
use std::fmt;
use std::fmt::{Formatter};
use serde::{Serialize, Deserialize};

#[derive(Debug, Serialize, Deserialize)]
pub struct VersionInfo {
    pub(crate) dependencies: HashMap<String, String>,
    pub(crate) timestamp: String
}

#[derive(Debug, Serialize, Deserialize)]
pub struct PackageInfo {
    pub(crate) versions: HashMap<String, VersionInfo>,
    pub(crate) name: String
}

#[derive(Debug)]
pub struct Node {
    pub(crate) name: String,
    pub(crate) version: String,
    pub(crate) timestamp: String,
}

impl fmt::Display for Node {
    fn fmt(&self, f: &mut Formatter<'_>) -> fmt::Result {
        write!(f, "{}-{}", self.name, self.version)
    }
}