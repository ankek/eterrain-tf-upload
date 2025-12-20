# Company Overview

## Company Information

**eTerrain** is a company that develops Carbon Management Platform (CMP) solutions for organizations seeking to track, measure, and manage their carbon footprint.

## Product Portfolio

### Carbon Management Platform (CMP)

The Carbon Management Platform (CMP) is the primary product in eTerrain's business portfolio, providing comprehensive carbon accounting and sustainability management solutions.

#### Product Components

The CMP consists of the following components:

1. **Data storage**
   - **Source Code Location**: no directory, only MySQL
   - **Purpose**: Data storage for all CMP components
   - **Target Users**: Field workers, mobile data entry, on-the-go monitoring
   - **Description**: the only data storage for eTerrain platform and its components
   - **Status**: Under development.

2. **Web Application**
   - **Source Code Location**: `eterrain-webapp` directory
   - **Purpose**: Primary web-based interface for carbon management by Organization users:
     * to track their carbon footprint
     * for managing organization settings
     * for generation reports and its configurations
   - **Target Users**: Desktop and tablet users, organization administrators
   - **Description**: A Go-based web application for managing carbon footprint data and environmental governance across multiple organizations. Each organization has its own configuration stored in MySQL DB with customizable feature flags and settings.
   - **Status**: Under development.

3. **Web Application**
   - **Source Code Location**: `eterrain-app-manager` directory
   - **Purpose**: Administration inferface of Carbon Management Platform (CMP) for management of Global configuration parameters (including eterrain-webapp) stored in My SQL database "data".
   - **Target Users**: Global admininstartor and apllication owners
   - **Description**: The eTerrain App Manager is a web-based portal for managing application owners (global administrators) of the eTerrain platform. Global configuration management with data stored in 'data' DB.
   - **Status**: MVP Developed. 

4. **Upload service**
   - **Source Code Location**: `eterrain-tf-upload` directory
   - **Purpose**: Data upload inferface of Carbon Management Platform (CMP) for processing uploaded data to be stored in MySQL Organization DB.
   - **Target Users**: Global admininstartor and apllication owners
   - **Description**: Backend service used for uploading Terrafom configurations of Organizations in theis MySQL DB. Part of Organization IaC and provide fully automated workflow to track their cloud resources
   - **Status**: MVP Developed. 

5. **Android Mobile Application**
   - **Source Code Location**: `eterrain-android-app` directory
   - **Purpose**: Mobile interface for carbon data collection and monitoring
   - **Target Users**: Field workers, mobile data entry, on-the-go monitoring
   - **Description**: Android application used for uploading data, files and images to be processed within eterrain-webapp to be stored in MySQL Organization DB.
   - **Status**: Not in development. 

6. **iOS Mobile Application**
   - **Source Code Location**: `eterrain-ios-app` directory
   - **Purpose**: Mobile interface for carbon data collection and monitoring
   - **Target Users**: Field workers, mobile data entry, on-the-go monitoring
   - **Description**: iOS application used for uploading data, files and images to be processed within eterrain-webapp to be stored in MySQL Organization DB.
   - **Status**: Not in development. 

## Business Context

- **Company Name**: eTerrain
- **Primary Product**: Carbon Management Platform (CMP)
- **Business Focus**: Carbon footprint tracking and sustainability management
- **Technology Stack**: Multi-platform solution (web + mobile)
- **Market Position**: Enterprise carbon accounting solutions

eTerrain Carbon Management Platform enables organizations to track, measure, and manage their carbon footprint across all emission scopes with comprehensive reporting and analytics capabilities. The Carbon Management Platform (CMP) is the primary product in eTerrain's business portfolio, providing comprehensive carbon accounting and sustainability management solutions.

in this dir ./inputs we have list of requirements for development structurized this way:

./inputs/*md - general requirements for all web components like security, etc.

./inputs/eterrain-webapp/*md - requirements for eterrain-webapp
./inputs/eterrain-app-manager/*md - requirements for eterrain-app-manager
./inputs/mysql/*md - requirements for mysql
./inputs/eterrain-mobileapp/*md - requirements for eterrain-mobileapp

./inputs/
./inputs/eterrain-webapp/
./inputs/eterrain-app-manager/
./inputs/mysql/
./inputs/eterrain-mobileapp/