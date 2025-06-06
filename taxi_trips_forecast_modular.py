
import pandas as pd

def create_features(df):
    df['trip_date'] = pd.to_datetime(df['trip_date'])
    df = df.sort_values(['zip_code', 'trip_date'])

    df_grouped = df.groupby('zip_code')
    for lag in [1, 7, 14]:
        df[f'lag_{lag}'] = df_grouped['trip_count'].shift(lag)

    for window in [7, 14]:
        df[f'rolling_mean_{window}'] = df_grouped['trip_count'].shift(1).rolling(window).mean()
        df[f'rolling_std_{window}'] = df_grouped['trip_count'].shift(1).rolling(window).std()

    df['dayofweek'] = df['trip_date'].dt.dayofweek
    df['is_weekend'] = df['dayofweek'].isin([5, 6]).astype(int)
    df['month'] = df['trip_date'].dt.month
    df['day'] = df['trip_date'].dt.day
    df['is_month_start'] = df['trip_date'].dt.is_month_start.astype(int)
    df['is_month_end'] = df['trip_date'].dt.is_month_end.astype(int)
    df['prior_week_avg'] = df_grouped['trip_count'].shift(7).rolling(7).mean()

    return df


import pandas as pd
import numpy as np
from sklearn.preprocessing import LabelEncoder
from xgboost import XGBRegressor
import optuna
import joblib
import os

def train_xgb_model(df, save_path="saved_xgb_model.json"):
    df = create_features(df)
    df = df.dropna().reset_index(drop=True)

    le = LabelEncoder()
    df['zip_code_encoded'] = le.fit_transform(df['zip_code'])

    features = [
        'zip_code_encoded', 'lag_1', 'lag_7', 'lag_14',
        'rolling_mean_7', 'rolling_std_7',
        'rolling_mean_14', 'rolling_std_14',
        'dayofweek', 'is_weekend',
        'month', 'day', 'is_month_start', 'is_month_end',
        'prior_week_avg'
    ]
    target = 'trip_count'

    def objective(trial):
        params = {
            "n_estimators": 1000,
            "max_depth": trial.suggest_int("max_depth", 3, 10),
            "learning_rate": trial.suggest_float("learning_rate", 0.01, 0.3),
            "subsample": trial.suggest_float("subsample", 0.6, 0.8),
            "colsample_bytree": trial.suggest_float("colsample_bytree", 0.6, 0.8),
            "min_child_weight": trial.suggest_int("min_child_weight", 1, 20),
            "gamma": trial.suggest_float("gamma", 0, 20),
            "reg_lambda": trial.suggest_float("reg_lambda", 0, 10),
            "reg_alpha": trial.suggest_float("reg_alpha", 0, 10),
            "objective": "reg:squarederror",
            "verbosity": 0
        }

        train_size = int(len(df) * 0.7)
        val_size = int(len(df) * 0.15)
        train = df.iloc[:train_size]
        val = df.iloc[train_size: train_size + val_size]

        model = XGBRegressor(**params)
        model.fit(train[features], train[target])
        preds = model.predict(val[features])
        return np.mean(np.abs(val[target] - preds))

    study = optuna.create_study(direction="minimize")
    study.optimize(objective, n_trials=30)

    best_params = study.best_params
    best_params.update({
        "n_estimators": study.best_trial.number,
        "objective": "reg:squarederror",
        "verbosity": 0
    })

    model = XGBRegressor(**best_params)
    model.fit(df[features], df[target])


    save_dir = os.path.join(os.getcwd(), 'Downloads')
    os.makedirs(save_dir, exist_ok=True)
    model.save_model(os.path.join(save_dir, save_path))
    joblib.dump(le, os.path.join(save_dir, 'label_encoder.pkl'))

    print(f"Model: {os.path.join(save_dir, save_path)}")
    print(f"Encoder: {os.path.join(save_dir, 'label_encoder.pkl')}")

    return model, le, features, df


import pandas as pd
import numpy as np
from xgboost import XGBRegressor
from dateutil.relativedelta import relativedelta
import joblib
def forecast_future(df, model_path="./saved_xgb_model.json", le_path="./label_encoder.pkl", output_path="./forecast_by_zipcode_daily_weekly_monthly.csv"):
    model = XGBRegressor()
    model.load_model(model_path)
    le = joblib.load(le_path)

    df = create_features(df)
    df = df.dropna().reset_index(drop=True)

    future_days, future_weeks, future_months = 30, 12, 6
    last_date = pd.to_datetime(df['trip_date'].max())
    zipcodes = df['zip_code'].unique()

    daily, weekly, monthly = [], [], []
    for z in zipcodes:
        for i in range(1, future_days + 1):
            daily.append({"trip_date": last_date + pd.Timedelta(days=i), "zip_code": z})
        for i in range(1, future_weeks + 1):
            weekly.append({"trip_date": last_date + pd.Timedelta(weeks=i), "zip_code": z})
        for i in range(1, future_months + 1):
            monthly.append({"trip_date": last_date + relativedelta(months=i), "zip_code": z})

    future_all = pd.concat([
        pd.DataFrame(daily).assign(freq='daily'),
        pd.DataFrame(weekly).assign(freq='weekly'),
        pd.DataFrame(monthly).assign(freq='monthly')
    ], ignore_index=True)

    future_all['dayofweek'] = future_all['trip_date'].dt.dayofweek
    future_all['is_weekend'] = future_all['dayofweek'].isin([5, 6]).astype(int)
    future_all['month'] = future_all['trip_date'].dt.month
    future_all['day'] = future_all['trip_date'].dt.day
    future_all['is_month_start'] = future_all['trip_date'].dt.is_month_start.astype(int)
    future_all['is_month_end'] = future_all['trip_date'].dt.is_month_end.astype(int)
    future_all['zip_code_encoded'] = le.transform(future_all['zip_code'])

    lag_features = [
        'lag_1', 'lag_7', 'lag_14',
        'rolling_mean_7', 'rolling_std_7',
        'rolling_mean_14', 'rolling_std_14',
        'prior_week_avg'
    ]

    latest_feats = (
        df.sort_values(['zip_code', 'trip_date'])
          .groupby('zip_code')
          .tail(1)[['zip_code'] + lag_features]
    )
    future_all = future_all.merge(latest_feats, on='zip_code', how='left')

    feature_cols = [
        'zip_code_encoded', 'lag_1', 'lag_7', 'lag_14',
        'rolling_mean_7', 'rolling_std_7',
        'rolling_mean_14', 'rolling_std_14',
        'dayofweek', 'is_weekend',
        'month', 'day', 'is_month_start', 'is_month_end',
        'prior_week_avg'
    ]

    future_all['trip_count_pred'] = model.predict(future_all[feature_cols])

    return future_all


import pandas as pd
from sqlalchemy import create_engine

username = 'username'
password = 'password'
dbname = 'dbname'

engine = create_engine(f'postgresql://{username}:{password}@localhost:5432/{dbname}')
query = "SELECT * FROM cur_zipcode_traffic_patterns"
df = pd.read_sql(query, engine)

# Train model
model, le, features, df_with_features = train_xgb_model(df)

# Forecast
forecast_df = forecast_future(df_with_features)
forecast_df.head()


# In[ ]:




