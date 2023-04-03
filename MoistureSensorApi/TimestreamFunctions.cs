using System.Text.Json;
using System.Text.Json.Serialization;
using Amazon;
using Amazon.Runtime;
using Amazon.Runtime.CredentialManagement;
using Amazon.TimestreamQuery;
using Amazon.TimestreamQuery.Model;

namespace MoistureSensorApi;

public class TimestreamFunctions
{
    // create an aws iot client
    private static readonly CredentialProfileStoreChain Chain = new();
    private static readonly AWSCredentials Credentials = null!;
    private static bool _ = Chain.TryGetAWSCredentials("aws-osuapp", out Credentials);
    
    // check if in development
    private static readonly bool InDevelopment = Environment.GetEnvironmentVariable("ASPNETCORE_ENVIRONMENT") == "Development";
    
    // create a timestream client
    private static readonly AmazonTimestreamQueryConfig TimestreamQueryClientConfig = new AmazonTimestreamQueryConfig 
    { 
        RegionEndpoint = RegionEndpoint.USWest2 
    };
    private static readonly AmazonTimestreamQueryClient TimestreamClient = InDevelopment ? new AmazonTimestreamQueryClient(Credentials, TimestreamQueryClientConfig) : new AmazonTimestreamQueryClient(TimestreamQueryClientConfig);
    
    // cancel query values
    private static readonly long ONE_GB_IN_BYTES = 1073741824L;
    private static readonly double QUERY_COST_PER_GB_IN_DOLLARS = 0.01; // Assuming the price of query is $0.01 per GB

    public static async Task<IEnumerable<string>> RunQueryAsync(string queryString)
    {
        var data = new string[] { };
        
        try
        {
            QueryRequest queryRequest = new QueryRequest();
            queryRequest.QueryString = queryString;
            QueryResponse queryResponse = await TimestreamClient.QueryAsync(queryRequest);
            while (true)
            {
                QueryStatus queryStatus = queryResponse.QueryStatus;
                double bytesMeteredSoFar = ((double) queryStatus.CumulativeBytesMetered / ONE_GB_IN_BYTES);
                // Cancel query if its costing more than 1 cent
                if (bytesMeteredSoFar * QUERY_COST_PER_GB_IN_DOLLARS > 0.01)
                {
                    await CancelQuery(queryResponse);
                    break;
                }
                
                var newData = ParseQueryResult(queryResponse);
                
                // merge newData to data
                data = data.Concat(newData).ToArray();
                
                if (queryResponse.NextToken == null)
                {
                    break;
                }
                queryRequest.NextToken = queryResponse.NextToken;
                queryResponse = await TimestreamClient.QueryAsync(queryRequest);
            }
        } catch(Exception e)
        {
            // Some queries might fail with 500 if the result of a sequence function has more than 10000 entries
            Console.WriteLine(e.ToString());
            
            throw;
        }
        
        // return the data array
        return data;
    }

    private static async Task CancelQuery(QueryResponse queryResponse)
    {
        Console.WriteLine("Cancelling query: " + queryResponse.QueryId);
        CancelQueryRequest cancelQueryRequest = new CancelQueryRequest();
        cancelQueryRequest.QueryId = queryResponse.QueryId;

        try
        {
            await TimestreamClient.CancelQueryAsync(cancelQueryRequest);
            Console.WriteLine("Query has been successfully cancelled.");
        } catch(Exception e)
        {
            Console.WriteLine("Could not cancel the query: " + queryResponse.QueryId + " = " + e);
        }
    }
    
    private static IEnumerable<string> ParseQueryResult(QueryResponse response)
    {
        List<ColumnInfo> columnInfo = response.ColumnInfo;
        var options = new JsonSerializerOptions
        {
            DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
        };
        List<String> columnInfoStrings = columnInfo.ConvertAll(x => JsonSerializer.Serialize(x, options));
        List<Row> rows = response.Rows;
        
        QueryStatus queryStatus = response.QueryStatus;
        Console.WriteLine("Current Query status:" + JsonSerializer.Serialize(queryStatus, options));
        
        Console.WriteLine("Metadata:" + string.Join(",", columnInfoStrings));
        Console.WriteLine("Data:");
        
        // create empty string array with no elements
        var data = new string[] { };

        foreach (Row row in rows)
        {
            Console.WriteLine(ParseRow(columnInfo, row));
            
            // append the row to the data array
            data = data.Append(ParseRow(columnInfo, row)).ToArray();
        }
        
        // return the data array
        return data;
    }

    private static string ParseRow(List<ColumnInfo> columnInfo, Row row)
    {
        List<Datum> data = row.Data;
        List<string> rowOutput = new List<string>();
        for (int j = 0; j < data.Count; j++)
        {
            ColumnInfo info = columnInfo[j];
            Datum datum = data[j];
            rowOutput.Add(ParseDatum(info, datum));
        }
        return $"{{{string.Join(",", rowOutput)}}}";
    }

    private static string ParseDatum(ColumnInfo info, Datum datum)
    {
        if (datum.NullValue)
        {
            return $"{info.Name}=NULL";
        }

        Amazon.TimestreamQuery.Model.Type columnType = info.Type;
        if (columnType.TimeSeriesMeasureValueColumnInfo != null)
        {
            return ParseTimeSeries(info, datum);
        }
        else if (columnType.ArrayColumnInfo != null)
        {
            List<Datum> arrayValues = datum.ArrayValue;
            return $"{info.Name}={ParseArray(info.Type.ArrayColumnInfo, arrayValues)}";
        }
        else if (columnType.RowColumnInfo != null && columnType.RowColumnInfo.Count > 0)
        {
            List<ColumnInfo> rowColumnInfo = info.Type.RowColumnInfo;
            Row rowValue = datum.RowValue;
            return ParseRow(rowColumnInfo, rowValue);
        }
        else
        {
            return ParseScalarType(info, datum);
        }
    }

    private static string ParseTimeSeries(ColumnInfo info, Datum datum)
    {
        var timeseriesString = datum.TimeSeriesValue
            .Select(value => $"{{time={value.Time}, value={ParseDatum(info.Type.TimeSeriesMeasureValueColumnInfo, value.Value)}}}")
            .Aggregate((current, next) => current + "," + next);

        return $"[{timeseriesString}]";
    }

    private static string ParseScalarType(ColumnInfo info, Datum datum)
    {
        return ParseColumnName(info) + datum.ScalarValue;
    }

    private static string ParseColumnName(ColumnInfo info)
    {
        return info.Name == null ? "" : (info.Name + "=");
    }

    private static string ParseArray(ColumnInfo arrayColumnInfo, List<Datum> arrayValues)
    {
        return $"[{arrayValues.Select(value => ParseDatum(arrayColumnInfo, value)).Aggregate((current, next) => current + "," + next)}]";
    }
}