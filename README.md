# NWU MSDS-432 : Chicago Business Intelligence for Strategic Planning

To setup and start the project following steps need to be followed:
git clone git@github.com:aishwaryamathuria/NWU_MSDS432_Project.git

## Postgres
  ```
  cd NWU_MSDS432_Project/docker_compose/postgres
  docker compose up
  ```
  This will bring a Postgres instance up and setup the database for Bronze, SIlver and Gold Tier.
  
## ETL
  
  Install docker
  ```
  cd NWU_MSDS432_Project/docker_compose/jenkins
  docker compose up
  docker ps
  docker cp -R ./jobs <replace with jenkins container id>:/var/jenkins_home
  ```
  Next create a new node in jenkins **"ETL_Baremetal"** and connect the machine by following instructions in the node details or https://www.jenkins.io/doc/book/using/using-agents/
  Once the machine is connected via agent.jar, run the jenkins jobs in the **"Bronze Tier"** folder to pull the data from the Chicago data SOAP API endpoints.

## Forecasting Microservice

  Install Python
  ```
  cd NWU_MSDS432_Project/be/forecasting-service
  pip install -r requirements.txt
  python covid_case_modler.py
  python traffic_pattern_modler.py
  python flask_app.py
  ```
  This will train the forecasting models from the data in the database and start an API service for prediction.

## Dashboard Microservice
  ```
  cd NWU_MSDS432_Project/docker_compose/Go_microservice
  docker compose up
  ```
  This will host the GO microservice endpoints for pulling the data from the database for different reports.

## UI Dashboard
  Install nvm and Node (18)
  ```
  cd NWU_MSDS432_Project/ui/report_dashboard
  npm run dev
  ```
  To build the UI for hosting on CDN 
  ```
  cd NWU_MSDS432_Project/ui/report_dashboard
  npm run build
  ```
  Copy the files in dist folder to a CDN for hosting.
