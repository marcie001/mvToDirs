package main

import(
  "flag"
  "fmt"
  "io"
  "io/ioutil"
  "log"
  "os"
  "path/filepath"
  "strings"
)

func main() {

  r := flag.Bool("r", false, "Scan files recursively")
  s := flag.String("s", "", "Source directory. (Required)")
  d := flag.String("d", "", "Destination directory. (Required)")
  flag.Parse()

  if s == nil || d == nil {
    flag.PrintDefaults()
    os.Exit(3)
  }
  if stat, err := os.Stat(*d); err != nil {
    log.Fatal(err)
  } else if !stat.IsDir() {
    log.Fatal(*d + " is not directory.")
  }

  if *r {
    if err := MvFilesR(*s, *d); err != nil {
      log.Fatal(err)
    }
  } else {
    if err := MvFiles(*s, *d); err != nil {
      log.Fatal(err)
    }
  }
}

// info の移動先ディレクトリを決定する。
// 移動先ディレクトリは "拡張子を小文字にした文字列/更新年月日" のような文字列で返される
// ただし、拡張子がない場合は "noext/更新年月日" という文字列を返す
func resolveDestDir(info os.FileInfo) string {
  ext := filepath.Ext(info.Name())
  if len(ext) == 0 {
    ext = "noext"
  } else {
    ext = strings.TrimLeft(ext, ".")
  }
  ts := info.ModTime()
  return filepath.Join(
      strings.ToLower(ext),
      fmt.Sprintf("%d%02d%02d", ts.Year(), ts.Month(), ts.Day()))
}

// 再帰的にファイル探し移動する
func MvFilesR(src string, dest string) error {
  return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }
    if info.Mode().IsRegular() {
      return Mv(path, info, dest)
    }
    return nil
  })
}

// ディレクトリ直下のファイルを移動する 
func MvFiles(src string, dest string) error {
  if info, err := os.Lstat(src); err != nil {
    return err
  } else if info.Mode().IsRegular() {
    return Mv(src, info, dest)
  } else if info.IsDir() {
    if fis, err := ioutil.ReadDir(src); err != nil {
      return err
    } else {
      for _, fi := range fis {
        if fi.Mode().IsRegular() {
          if err = Mv(filepath.Join(src, fi.Name()), fi, dest); err != nil {
            return err
          }
        }
      }
    }
  }
  return nil
}

// srcPath のファイルを移動する。上書きはしない
func Mv(srcPath string, srcInfo os.FileInfo, dest string) error {
  destroot := filepath.Join(dest, resolveDestDir(srcInfo))
  if _, err := os.Stat(destroot); err != nil {
    // permision に 0777 をしているが umask は考慮される
    if err := os.MkdirAll(destroot, 0777); err != nil {
      return err
    }
  }
  destPath := filepath.Join(destroot, srcInfo.Name())
  if f, _ := os.Stat(destPath); f != nil {
    fmt.Printf("%s exsits. %s didn't move.\n", destPath, srcPath)
    return nil
  }
  err := os.Rename(srcPath, destPath)
  switch err.(type) {
  case *os.LinkError:
    if e := Cp(srcPath, destPath); e != nil {
      return e
    }
    if e := os.Remove(srcPath); e != nil {
      return e
    }
  default:
    return err
  }
  return nil
}

func Cp(src string, dest string) error {
  reader, e := os.Open(src)
  if e != nil {
    return e
  }
  defer reader.Close()

  writer, e := os.Create(dest)
  if e != nil {
    return e
  }
  defer writer.Close()

  if _, e = io.Copy(writer, reader); e != nil {
    return e
  }
  return nil
}
