using System;
using System.IO;
using System.Threading;
using OpenCvSharp;
using SkiaSharp;
using System.Collections.Concurrent;
using System.IO.Compression;
using System.Threading.Tasks;
using System.Collections.Generic;

namespace OCRTrainingDataMakerSharp
{
    class ImageFile
    {
        public string Filename;
        public SKImage Data;
    }
    class Program
    {
        public static BlockingCollection<ImageFile> Images = new BlockingCollection<ImageFile>(1000);
        public static BlockingCollection<int> ThCount = new BlockingCollection<int>(20);
        public static List<string> Fonts = new List<string>();
        public static Dictionary<int, string> dic = new Dictionary<int, string>();
        public static bool RunFlag = true;
        public static Int64 FileCount=0;
        public static DateTime startTime = DateTime.Now;
        static void Main(string[] args)
        {
            
            DirectoryInfo Dir = new DirectoryInfo("./fonts");
            foreach (var fileinfo in Dir.GetFiles())
            {
                if (fileinfo.FullName.ToLower().Contains("ttf")|| fileinfo.FullName.ToLower().Contains("otf"))
                {
                    Fonts.Add(fileinfo.FullName);
                }
            }

            FileStream fs = new FileStream("./charlist.txt", FileMode.Open, FileAccess.Read);
            StreamReader reader = new StreamReader(fs);
            reader.BaseStream.Seek(0, SeekOrigin.Begin);
            string strLine = reader.ReadLine();
            var ccount = 1;
            while (strLine != null)
            {
                dic.Add(ccount, strLine.ToCharArray()[0].ToString());
                ccount++;
                strLine = reader.ReadLine();
            }
            var StatThread = new Thread(new ThreadStart(Stat));
            StatThread.Start();
            var SaveThread = new Thread(new ThreadStart(SaveImage));
            SaveThread.Start();
            var i = 0;
            foreach (var id in dic.Keys){
                i++;
                ThCount.Add(1);
                Console.WriteLine("Processing[{0}/{1}]{2}-{3}", i, dic.Count, id, dic[id]);
                var t= new Thread(new ParameterizedThreadStart(ProcessImg));
                t.Start(i.ToString());
                //ProcessImg(id);
            }
            while(ThCount.Count>0 || Images.Count>0){
                Thread.Sleep(1000);
            }
            RunFlag = false;
        }

        public static void ProcessImg(object obj)
        {
            var id = int.Parse(obj.ToString());
            var str = dic[id];
            var skFontPaint = new SKPaint
            {
                TextSize = 100 - 10,
                Color = SKColors.White,
                TextAlign = SKTextAlign.Center
            };
            foreach (var font in Fonts)
            {
                skFontPaint.Typeface = SKTypeface.FromFile("sarasa-gothic-sc-regular.ttf");
                var count = 0;
                SKFontMetrics sKFontMetrics;
                skFontPaint.GetFontMetrics(out sKFontMetrics);

                var skSurface = SKSurface.Create(new SKImageInfo(100, 100));
                var skCanvas = skSurface.Canvas;
                skCanvas.DrawText(str, 50, 100 - 10 - (int)sKFontMetrics.UnderlinePosition, skFontPaint);

                var image = skSurface.Snapshot();
                skCanvas.Dispose();
                skSurface.Dispose();

                for (float degree = -90; degree <= 90; degree++)
                {
                    var img = ApplyRotate(image, degree);
                    Images.Add(new ImageFile { Filename = id.ToString() + "-" + str + "-" + count.ToString() + ".png", Data = img });
                    count++;
                    var imge = ApplyErode(img, 3);
                    Images.Add(new ImageFile { Filename = id.ToString() + "-" + str + "-" + count.ToString() + ".png", Data = imge });
                    count++;
                    Images.Add(new ImageFile { Filename = id.ToString() + "-" + str + "-" + count.ToString() + ".png", Data = ApplyErode(img, 3) });
                    count++;
                    Images.Add(new ImageFile { Filename = id.ToString() + "-" + str + "-" + count.ToString() + ".png", Data = ApplyErode(imge, 3) });
                    count++;
                }
            }
            ThCount.Take();
        }
        public static void Stat()
        {
            while(RunFlag){
                Console.WriteLine("Speed:{0}per sec", FileCount / ((int)(DateTime.Now - startTime).TotalSeconds + 1));
                Thread.Sleep(1000);
            }
        }
        public static void SaveImage()
        {
            //var zipMs = new MemoryStream();
            if (File.Exists($"./Output.zip"))
            {
                File.Delete($"./Output.zip");
            }
            StreamWriter Tw = new StreamWriter("./Train.txt");
            StreamWriter Vw = new StreamWriter("./val.txt");
            FileStream fsWrite = new FileStream($"./Output.zip", FileMode.Create);
            var zipArchive = new ZipArchive(fsWrite, ZipArchiveMode.Create);
            while (RunFlag)
            {
                ImageFile image;
                if (Images.TryTake(out image, 1000) == true)
                {
                    FileCount++;
                    var zipFileStream = zipArchive.CreateEntry(image.Filename, CompressionLevel.NoCompression).Open();
                    image.Data.Encode().SaveTo(zipFileStream);
                    zipFileStream.Close();
                    var Count = int.Parse(image.Filename.Split(".")[0].Split("-")[2]);
                    if (Count % 5 == 0)
                    {
                        Vw.WriteLine(image.Filename + " " + image.Filename.Split(".")[0].Split("-")[1]);
                        Vw.Flush();
                    }
                    else
                    {
                        Tw.WriteLine(image.Filename + " " + image.Filename.Split(".")[0].Split("-")[1]);
                        Tw.Flush();
                    }
                }
            }
            //File.WriteAllBytes($"./Output.zip", zipMs.ToArray());
            zipArchive.Dispose();
            fsWrite.Close();
            Tw.Close();
            Vw.Close();
        }

        public static SKImage ApplyRotate(SKImage image, float degree)
        {
            var skSurface = SKSurface.Create(new SKImageInfo(100, 100));
            var skCanvas = skSurface.Canvas;
            skCanvas.Clear();
            skCanvas.RotateDegrees(degree, 50, 50);
            skCanvas.DrawImage(image, 0, 0);
            var img = skSurface.Snapshot();
            skCanvas.Dispose();
            skSurface.Dispose();
            return img;
        }

        public static SKImage ApplyErode(SKImage image, int degree)
        {
            var skErode = SKImageFilter.CreateErode(degree, degree);
            var skSurface = SKSurface.Create(new SKImageInfo(100, 100));
            var skCanvas = skSurface.Canvas;
            var Paint = new SKPaint();
            Paint.ImageFilter = skErode;
            skCanvas.Clear();
            skCanvas.DrawImage(image, 0, 0, Paint);
            var img = skSurface.Snapshot();
            skCanvas.Dispose();
            skSurface.Dispose();
            return img;
        }

        public static SKImage ApplyDilate(SKImage image, int degree)
        {
            var skErode = SKImageFilter.CreateDilate(degree, degree);
            var skSurface = SKSurface.Create(new SKImageInfo(100, 100));
            var skCanvas = skSurface.Canvas;
            var Paint = new SKPaint();
            Paint.ImageFilter = skErode;
            skCanvas.Clear();
            skCanvas.DrawImage(image, 0, 0, Paint);
            var img = skSurface.Snapshot();
            skCanvas.Dispose();
            skSurface.Dispose();
            Paint.Dispose();
            skErode.Dispose();
            return img;
        }

        public static SKImage ApplyNoise(SKImage image)
        {
            var skSurface = SKSurface.Create(new SKImageInfo(100, 100));
            var skCanvas = skSurface.Canvas;
            skCanvas.Clear();
            var skPaint = new SKPaint();

            if (StaticRandom.Instance.NextDouble() < 0.5)
            {
                skPaint.ImageFilter = SKImageFilter.CreateDilate(StaticRandom.Instance.Next(0, 2), StaticRandom.Instance.Next(0, 2));
            }
            else
            {
                skPaint.ImageFilter = SKImageFilter.CreateErode(StaticRandom.Instance.Next(0, 3), StaticRandom.Instance.Next(0, 3));
            }
            if (StaticRandom.Instance.NextDouble() < 0.5)
            {
                var noiseSize = 3;
                var noiseCount = 20;
                for (var i = 0; i < noiseCount; i++)
                {
                    var x = StaticRandom.Instance.Next(0, 100 - noiseSize);
                    var y = StaticRandom.Instance.Next(0, 100 - noiseSize);

                    skCanvas.DrawRect(new SKRect(x, y, x + noiseSize, y + noiseSize), new SKPaint { Color = SKColors.White });
                }
            }
            skCanvas.DrawImage(image, 0, 0, skPaint);
            var img = skSurface.Snapshot();
            skCanvas.Dispose();
            skSurface.Dispose();
            return img;
        }
        static class StaticRandom
        {
            private static int seed;

            private static ThreadLocal<Random> threadLocal = new ThreadLocal<Random>
                (() => new Random(Interlocked.Increment(ref seed)));

            static StaticRandom()
            {
                seed = Environment.TickCount;
            }

            public static Random Instance { get { return threadLocal.Value; } }
        }
    }

}
